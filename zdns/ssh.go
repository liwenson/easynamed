package zdns

import (
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"path"
)

var (
	sshClient  *ssh.Client
	sftpClient *sftp.Client
)

// 执行远程命令
func exec_cmd(cmd string) (outStr string, exitcode int) {
	session, err := sshClient.NewSession()
	if err != nil {
		panic(err)
	}

	defer session.Close()

	log.Print(cmd)
	buf, errStr := session.CombinedOutput(cmd)
	if errStr != nil {

		if ins, ok := errStr.(*ssh.ExitError); ok {
			outStr = string(buf)
			exitcode = ins.ExitStatus()
			return outStr, exitcode
		}

	}

	return string(buf), 0
}

// 上传文件
func pushFile(localpath, remotepath, remoteFileName string) bool {

	// 打开本地文件
	sourceFile, err := os.Open(localpath)
	if err != nil {
		panic(err)
	}

	fmt.Printf("----==>>>> %s/%s\n", remotepath, remoteFileName)

	targetFile, e := sftpClient.Create(path.Join(remotepath, remoteFileName))

	//dstFile, e := sftpClient.Create(remoteFileName)
	if e != nil {
		log.Println(errors.New("Could not create the remote file."))
		log.Fatal(e.Error())
		return false
	}

	buffer := make([]byte, 1024)
	for {
		n, err := sourceFile.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				fmt.Println("读取文件出错", err)
				log.Fatal(err)
			}
		}
		targetFile.Write(buffer[:n])
		//注意，由于文件大小不定，不可直接使用buffer，否则会在文件末尾重复写入，以填充1024的整数倍
	}

	defer func() {
		targetFile.Close()
		sftpClient.Close()
		sourceFile.Close()
	}()

	log.Print("File push succeeded")

	return true
}
