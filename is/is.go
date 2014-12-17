package main

import (
	"encoding/json"
	"errors"
	"fmt"
	//"golang.org/x/crypto/ssh"
	//"golang.org/x/crypto/ssh/agent"
	"io/ioutil"
	"log"
	//"net"
	"crypto/md5"
	"flag"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

const (
	ChunkSize = 128 * 1024 * 1024
)

type TransferConfig struct {
	SrcUser    string
	SrcAddr    string
	SrcPath    string
	DstUser    string
	DstAddr    string
	DstPath    string
	DstThrough []string
}

/*
func DialWithAgentForwarded(user, addr string) (*ssh.Session, error) {
	agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, err
	}
	defer agentConn.Close()

	agentClient := agent.NewClient(agentConn)
	auths := []ssh.AuthMethod{ssh.PublicKeysCallback(agentClient.Signers)}

	config := &ssh.ClientConfig{
		User: user,
		Auth: auths}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	agent.ForwardToRemote(client, os.Getenv("SSH_AUTH_SOCK"))

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	err = agent.RequestAgentForwarding(session)
	if err != nil {
		session.Close()
		return nil, err
	}

	return session, nil
}
func Transfer(config *TransferConfig) error {
	session, err := DialWithAgentForwarded(config.SrcUser, config.SrcAddr+":22")
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	cmd := fmt.Sprintf("scp -o StrictHostKeyChecking=no \"%s\" \"%s@%s:%s\"",
		config.SrcPath, config.DstUser, config.DstAddr, config.DstPath)
	log.Printf("Executing remote command: %s\n", cmd)
	return session.Run(cmd)
}
*/

/*
func EncodeSshConfig() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	bytes, err := ioutil.ReadFile(usr.HomeDir + "/.ssh/config")
	if err != nil {
		return "", nil
	}

	result := ""
	for _, line := range strings.Split(string(bytes), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		result += fmt.Sprintf("-o \"%s\" ", trimmed)
	}
	return result, nil
}
*/

func Encode(s string) string {
	result := ""
	for _, c := range s {
		if c == '\\' {
			result += "\\\\"
		} else if c == '"' {
			result += "\\\""
		} else {
			result += string(c)
		}
	}
	return result
}

func ConvertProxyCommand(through []string) string {
	if len(through) == 0 {
		return ""
	} else {
		return Encode("-o \"ProxyCommand=ssh " + ConvertProxyCommand(through[1:]) + through[0] + " nc %h %p\" ")
	}
}

func Transfer(config *TransferConfig) error {
	cmdStr := fmt.Sprintf("ssh -A %s@%s scp -o StrictHostKeyChecking=no %s\"%s\" \"%s@%s:%s\"",
		config.SrcUser, config.SrcAddr, ConvertProxyCommand(config.DstThrough), config.SrcPath,
		config.DstUser, config.DstAddr, config.DstPath)
	cmd := exec.Command("/bin/sh", "-xc", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Scp(user, addr, from, to string) error {
	cmdStr := fmt.Sprintf("scp %s %s@%s:%s", from, user, addr, to)
	cmd := exec.Command("/bin/sh", "-xc", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ChunkedTransfer(config *TransferConfig) error {
	// 1. Send IS command to the source host
	err := Scp(config.SrcUser, config.SrcAddr, "./is", "/tmp/is")
	if err != nil {
		return err
	}
	// 2. Run create-catalog on the source host
	// 3. Send IS comamnd to the destination host
	err = Scp(config.DstUser, config.DstAddr, "./is", "/tmp/is")
	if err != nil {
		return err
	}
	// 4. Run create-catalog on the destination host
	// 5. Take diff on user client and send (parallel) scp command(s) to the source host
	// 6. Run patch-by-catalog on the destination host
	return nil
}

type HostConfig struct {
	User    string
	Host    string
	BaseDir string
	Through []string
}

type HostConfigs map[string]HostConfig

func GetHostConfigs() *HostConfigs {
	usr, err := user.Current()
	if err != nil {
		log.Fatalln(err)
	}

	bytes, err := ioutil.ReadFile(usr.HomeDir + "/.isrc")
	if err != nil {
		log.Fatalln(err)
	}

	result := make(HostConfigs)

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		log.Fatalln(err)
	}

	return &result
}

func SplitLocation(location string) (host string, dir string, err error) {
	splitted := strings.Split(location, ":")
	if len(splitted) != 2 {
		err = errors.New(fmt.Sprintf("invalid location %s", location))
		return
	}

	host = splitted[0]
	dir = splitted[1]
	return
}

func GetFullLocation(hostAlias, relPath string, configs *HostConfigs) (user string, addr string, path string, through []string, err error) {
	config, ok := (*configs)[hostAlias]
	if !ok {
		err = errors.New(fmt.Sprintf("invalid host name alias %s", hostAlias))
		return
	}

	user = config.User
	addr = config.Host
	path = config.BaseDir + "/" + relPath
	through = config.Through
	return
}

func DoTransfer(argSrc, argDst string, configs *HostConfigs, chunked bool) error {
	srcHost, src, err := SplitLocation(argSrc)
	if err != nil {
		return err
	}

	dstHost, dst, err := SplitLocation(argDst)
	if err != nil {
		return err
	}

	transferConfig := new(TransferConfig)
	usr, addr, path, _, err := GetFullLocation(srcHost, src, configs)
	if err != nil {
		return err
	}
	transferConfig.SrcUser = usr
	transferConfig.SrcAddr = addr
	transferConfig.SrcPath = path

	usr, addr, path, through, err := GetFullLocation(dstHost, dst, configs)
	if err != nil {
		return err
	}
	transferConfig.DstUser = usr
	transferConfig.DstAddr = addr
	transferConfig.DstPath = path
	transferConfig.DstThrough = through

	if chunked {
		err = ChunkedTransfer(transferConfig)
	} else {
		err = Transfer(transferConfig)
	}
	if err != nil {
		return err
	}

	return nil
}

type Catalog struct {
	Size     int
	Metadata []ChunkMetadata
}

type ChunkMetadata struct {
	Begin int // [begin, end)
	End   int
	Hash  string
}

func CreateCatalog(fileName string, expectedFileSize int) *Catalog {
	if expectedFileSize != -1 {
		info, err := os.Stat(fileName)
		if err != nil {
			log.Fatalln(err)
		}
		actualFileSize := int(info.Size())

		if expectedFileSize != actualFileSize {
			rem := expectedFileSize
			file, err := os.Create(fileName)
			if err != nil {
				log.Fatalln(err)
			}

			bytes := make([]byte, ChunkSize)
			for rem > 0 {
				cur := 0
				if rem < ChunkSize {
					cur = rem
				} else {
					cur = ChunkSize
				}
				n, err := file.Write(bytes[0:cur])
				if err != nil {
					log.Fatalln(err)
					file.Close()
				}
				rem -= n
			}
			file.Close()
		}
	}

	info, err := os.Stat(fileName)
	if err != nil {
		log.Fatalln(err)
	}
	actualFileSize := int(info.Size())

	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	total := 0

	catalog := Catalog{Size: actualFileSize}
	catalog.Metadata = make([]ChunkMetadata, 0)

	for total < actualFileSize {
		bytes := make([]byte, ChunkSize)
		n, _ := file.Read(bytes)

		if n < ChunkSize && total != actualFileSize {
			log.Fatalln("invalid state")
		}

		hash := fmt.Sprintf("%x", md5.Sum(bytes))
		catalog.Metadata = append(catalog.Metadata, ChunkMetadata{Begin: total, End: total + n, Hash: hash})
		total += n
	}

	return &catalog
}

func DoCreateCatalog() {
	if flag.NArg() != 1 && flag.NArg() != 2 {
		log.Fatalln("invalid number of arguments for create-catalog mode")
	}

	fileName := flag.Arg(0)
	expectedFileSize := -1
	if flag.NArg() == 2 {
		var err error
		expectedFileSize, err = strconv.Atoi(flag.Arg(1))
		if err != nil {
			log.Fatalln(err)
		}
	}

	catalog := CreateCatalog(fileName, expectedFileSize)

	marshaled, err := json.Marshal(catalog)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(string(marshaled))
}

func DoPatchByCatalog() {
	if flag.NArg() != 1 {
		log.Fatalln("invalid number of arguments for patch-by-catalog mode")
	}

	readFromStdin, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalln(err)
	}

	var expectedCatalog Catalog
	err = json.Unmarshal(readFromStdin, &expectedCatalog)
	if err != nil {
		log.Fatalln(err)
	}

	fileName := flag.Arg(0)
	actualCatalog := CreateCatalog(fileName, -1)

	if len(expectedCatalog.Metadata) != len(actualCatalog.Metadata) {
		log.Fatalln("file size inconsistent")
	}

	metadataCount := len(expectedCatalog.Metadata)

	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	for i := 0; i < metadataCount; i++ {
		if expectedCatalog.Metadata[i].Begin != actualCatalog.Metadata[i].Begin ||
			expectedCatalog.Metadata[i].End != actualCatalog.Metadata[i].End {
			log.Fatalln("chunk size inconsistent")
		}

		bytes, err := ioutil.ReadFile("/tmp/" + expectedCatalog.Metadata[i].Hash)
		if err != nil {
			log.Fatalln(err)
		}
		n, err := file.WriteAt(bytes, int64(expectedCatalog.Metadata[i].Begin))
		if err != nil {
			log.Fatalln(err)
		}
		n = n
		// TODO: check n != End - Begin + 1
	}
}

func main() {
	flagChunked := flag.Bool("chunked", false, "do chunked transfer (experimental)")
	flagCreateCatalog := flag.Bool("create-catalog", false, "internal use only")
	flagPatchByCatalog := flag.Bool("patch-by-catalog", false, "internal use only")

	flag.Parse()

	if *flagCreateCatalog {
		DoCreateCatalog()
		return
	}

	if *flagPatchByCatalog {
		DoPatchByCatalog()
	}

	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr,
			`Usage: is [options] srcHost:src dstHost:dst
Options: 
	--chunked=false: do chunked transfer (experimental)

You have to define host configuration in ~/.isrc like this:
{
  "host1": {
    "User": "alice",
    "Host": "example.com",
    "BaseDir": "/data/foo"
  },
 "host2": {
    "User": "bob",
    "Host": "example.org",
    "BaseDir": "/home/foobar/baz",
    "Through": ["example.net", "example.com"]
  }
}
`)
		return
	}

	configs := GetHostConfigs()

	err := DoTransfer(flag.Args()[0], flag.Args()[1], configs, *flagChunked)
	if err != nil {
		log.Fatalln(err)
	}
}
