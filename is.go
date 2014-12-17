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
	"flag"
	"os"
	"os/exec"
	"os/user"
	"strings"
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

func DoTransfer(argSrc, argDst string, configs *HostConfigs) error {
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

	err = Transfer(transferConfig)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	if len(flag.NArg()) != 2 {
		fmt.Fprintln(os.Stderr,
			`Usage: is srcHost:src dstHost:dst

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

	err := DoTransfer(flag.Args()[0], flag.Args()[1], configs)
	if err != nil {
		log.Fatalln(err)
	}
}
