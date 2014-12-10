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
	"os"
	"os/exec"
	"os/user"
	"strings"
)

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
*/

type TransferConfig struct {
	SrcUser string
	SrcAddr string
	SrcPath string
	DstUser string
	DstAddr string
	DstPath string
}

/*
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
		result += fmt.Sprintf("-o \"%s\"", line)
	}
	return result, nil
}

func Transfer(config *TransferConfig) error {
	sshConfig, err := EncodeSshConfig()
	if err != nil {
		return err
	}

	cmd := exec.Command("/bin/sh", "-xc",
		fmt.Sprintf("ssh -A %s@%s scp -o StrictHostKeyChecking=no %s\"%s\" \"%s@%s:%s\"",
			config.SrcUser, config.SrcAddr, sshConfig, config.SrcPath,
			config.DstUser, config.DstAddr, config.DstPath))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type HostConfig struct {
	User    string
	Host    string
	BaseDir string
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

func GetFullLocation(hostAlias, relPath string, configs *HostConfigs) (user string, addr string, path string, err error) {
	config, ok := (*configs)[hostAlias]
	if !ok {
		err = errors.New(fmt.Sprintf("invalid host name alias %s", hostAlias))
		return
	}

	user = config.User
	addr = config.Host
	path = config.BaseDir + "/" + relPath
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
	usr, addr, path, err := GetFullLocation(srcHost, src, configs)
	if err != nil {
		return err
	}
	transferConfig.SrcUser = usr
	transferConfig.SrcAddr = addr
	transferConfig.SrcPath = path

	usr, addr, path, err = GetFullLocation(dstHost, dst, configs)
	if err != nil {
		return err
	}
	transferConfig.DstUser = usr
	transferConfig.DstAddr = addr
	transferConfig.DstPath = path

	err = Transfer(transferConfig)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	if len(os.Args) != 3 {
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
    "BaseDir": "/home/foobar/baz"
  }
}
`)
		return
	}

	configs := GetHostConfigs()

	err := DoTransfer(os.Args[1], os.Args[2], configs)
	if err != nil {
		log.Fatalln(err)
	}
}
