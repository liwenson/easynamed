package zdns

import (
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"text/template"
	"time"
)

type Zone struct {
	Name      string // 域
	Server    string // 域ip
	Operation string // 操作指令 add , del , pause , thaw , status
}

type Record struct {
	Name string // 记录名称
	Type string // DNS记录, A,CNAME,NS,SOA,AAAA,TXT,MX,PTR,SRV,URL
	Ttl  int64  // 生存时间，解析记录会在dns服务器中缓存的时间
	Addr string // 记录值
	Desc string // 描述
}

type Server struct {
	Host    string
	User    string
	Pwd     string
	Port    int
	ZoneDir string
}

type Zdns struct{}

var (
	server *Server
)

func NewZdns(s *Server) (zdns *Zdns) {
	server = s
	return
}

func (z *Zdns) Init(sshCli *ssh.Client) {
	sshClient = sshCli

	var err error
	sftpClient, err = sftp.NewClient(sshCli)
	if err != nil {
		log.Fatal(err)
	}

}

func (z *Zdns) Connect() (sshCli *ssh.Client, err error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
	)

	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(server.Pwd))

	hostKeyCallbk := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}

	clientConfig = &ssh.ClientConfig{
		User:            server.User,
		Auth:            auth,
		Timeout:         30 * time.Second,
		HostKeyCallback: hostKeyCallbk,
	}

	// connet to ssh
	addr = fmt.Sprintf("%s:%d", server.Host, server.Port)

	if sshCli, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	return
}

func (z *Zdns) Zone(zone *Zone) (string, bool) {

	operation := zone.Operation

	switch operation {
	case "add":
		if zone.Server == "" {
			zone.Server = server.Host
		}

		return "", z.zoneCreate(zone)

	case "del":
		return "", z.zoneDel(zone)
	case "pause":
		return "", z.zonePause(zone)
	case "thaw":
		return "", z.zoneThaw(zone)
	case "flush":
		return "", z.zoneFlush(zone)
	case "sync":
		return "", z.zoneSync(zone)
	case "reload":
		return "", z.zoneReload()
	case "status":
		return z.zoneStatus(zone)
	default:
		return "", false
	}

}

// ZoneReload Reload configuration file and zones.
func (z *Zdns) zoneReload() bool {

	cmd := "rndc reload"

	out, code := exec_cmd(cmd)
	if code != 0 {
		log.Fatal(out)
		return false
	}

	return true
}

// ZoneSync 将日志文件同步到区域文件
func (z *Zdns) zoneSync(zone *Zone) bool {

	var cmsStr string
	if zone.Name != "" {
		cmsStr = fmt.Sprintf("rndc sync -clean %s", zone.Name)
		out, code := exec_cmd(cmsStr)
		if code != 0 {
			log.Fatal(out)
			return false
		}
		return true
	}

	return false
}

// ZoneFlush 清除缓存
func (z *Zdns) zoneFlush(zone *Zone) bool {

	var cmsStr string
	if zone.Name != "" {
		cmsStr = fmt.Sprintf("rndc flushname %s", zone.Name)
		out, code := exec_cmd(cmsStr)
		if code != 0 {
			log.Fatal(out)
			return false
		}
		return true
	}

	return false
}

// ZoneStatus Reload configuration file and zones.
func (z *Zdns) zoneStatus(zone *Zone) (string, bool) {

	var cmsStr string
	if zone.Name != "" {
		cmsStr = fmt.Sprintf("rndc zonestatus %s", zone.Name)
		out, code := exec_cmd(cmsStr)
		if code != 0 {
			log.Fatal(out)
			return "", false
		}
		return out, true
	}

	return "", false
}

// ZoneCreate 创建 zone
func (z *Zdns) zoneCreate(zone *Zone) bool {
	// 创建临时文件，用来存放命令
	file, err := os.CreateTemp("", "zone.tmp")
	if err != nil {
		log.Print(err)
		return false
	}

	defer func() {
		file.Close()
		// 一般来说，临时文件不用了，需要移除
		err := os.Remove(file.Name())
		if err != nil {
			log.Fatal(err.Error())
		}
	}()

	templateStr := `$TTL 86400	; 1 day
@        IN SOA    dns.{{.Name}}.  admin.{{.Name}}.  (
				0         ; serial 
				10800      ; refresh (3 hours)
				900        ; retry (15 minutes)
				604800     ; expire (1 week)
				86400      ; minimum (1 day)
				)
@        IN    NS    dns.{{.Name}}.
dns      IN    A     {{.Server}}

`

	// 填充模板
	tmpl, err := template.New("zone").Parse(templateStr)
	err = tmpl.Execute(file, &zone)
	if err != nil {
		return false
	}

	// 传输
	zoneName := fmt.Sprintf("%s.zone", zone.Name)
	res := checkFIle(fmt.Sprintf("%s/%s", server.ZoneDir, zoneName))
	fmt.Println("res:", res)
	if !res {
		// 传输模板
		if !pushFile(file.Name(), server.ZoneDir, zoneName) {
			return false
		}
	}

	// 检查模板格式是否正确
	if checkzone(zoneName, zone.Name) {
		cmd := fmt.Sprintf("rndc addzone %s '{ type master; file \"zone/%s\"; allow-update{any;};};'", zone.Name, zoneName)
		//fmt.Println("cmd:", cmd)

		out, code := exec_cmd(cmd)
		if code != 0 {
			log.Printf(out)
			return false
		}
	}

	return true

}

func (z *Zdns) zoneDel(zone *Zone) bool {

	// 区域有效性检查
	log.Printf("删除zone %s", zone.Name)
	cmdStr := fmt.Sprintf("rndc delzone -clean %s ", zone.Name)
	out, code := exec_cmd(cmdStr)
	if code != 0 {
		log.Fatal(out)
		return false
	}

	return true

}

func (z *Zdns) zonePause(zone *Zone) bool {

	// 区域有效性检查
	log.Printf("暂停 zone %s", zone.Name)
	cmdStr := fmt.Sprintf("rndc freeze %s ", zone.Name)
	out, code := exec_cmd(cmdStr)
	if code != 0 {
		log.Fatal(out)
		return false
	}

	return true

}

func (z *Zdns) zoneThaw(zone *Zone) bool {

	// 区域有效性检查
	log.Printf("恢复 zone %s", zone.Name)
	cmdStr := fmt.Sprintf("rndc thaw %s ", zone.Name)
	out, code := exec_cmd(cmdStr)
	if code != 0 {
		log.Fatal(out)
		return false
	}

	return true

}

type Domain struct {
	Records   []*Record
	Zone      string
	Operation string // add ,del
}

// DomainOperation 添加，删除操作
// Operation   add,del
func (z *Zdns) DomainOperation(d *Domain) bool {
	// 生成临时文件
	//在dir目录下创建一个新的、使用prefix为前缀的临时文件，以读写模式打开该文件并返回os.File指针。如果dir是空字符串，
	//TempFile使用默认用于临时文件的目录（参见os.TempDir函数）。不同程序同时调用该函数会创建不同的临时文件，调用本函数的程序有责任在不需要临时文件时摧毁它。
	file, err := os.CreateTemp("", "records.temp")
	if err != nil {
		panic(err)
	}
	defer func() {
		file.Close()
		// 一般来说，临时文件不用了，需要移除
		err := os.Remove(file.Name())
		if err != nil {
			log.Fatal(err.Error())
		}
	}()

	// 标识限制
	var operation string
	if d.Operation == "del" {
		operation = "del"
	} else if d.Operation == "add" {
		operation = "add"
	} else {
		log.Fatal(errors.New("只接受 add,del 操作标识"))
		return false
	}

	// 写入模板
	sweaters := struct {
		Records   []*Record
		Server    string
		Zname     string
		Operation string
	}{Records: d.Records, Server: server.Host, Zname: d.Zone, Operation: operation}

	var domainTemplate = `server {{.Server}}
zone {{.Zname}}
{{if eq .Operation "del"}}
{{range $val := .Records}}
update delete {{$val.Name}}.{{$.Zname}} {{$val.Ttl}} {{$val.Type}}
{{end}}
{{else}}
{{range $val := .Records}}
update add {{$val.Name}}.{{$.Zname}} {{$val.Ttl}} {{$val.Type}} {{$val.Addr}}
{{end}}
{{end}}
send
`

	if tmpl, err := template.New("domain").Parse(domainTemplate); err != nil {
		log.Fatal(err)
	} else {
		err = tmpl.Execute(file, sweaters)
		if err != nil {
			log.Fatal(err)
		}

	}

	// 传输模板
	dstName := "nsupdate.tmp"
	if pushFile(file.Name(), "/tmp", dstName) {
		// 检查模板格式是否正确
		cmd := fmt.Sprintf("nsupdate -v /tmp/%s", dstName)

		out, code := exec_cmd(cmd)

		if code != 0 {
			log.Fatal(out)
			return false
		}

		defer func() {
			cmd := fmt.Sprintf("rm -f  /tmp/%s", dstName)
			out, code := exec_cmd(cmd)

			if code != 0 {
				log.Fatal(out)
			}
		}()
	} else {
		return false
	}
	return true
}

// checkzone zone配置检查
func checkzone(remoteFileName, zone string) bool {
	// 区域有效性检查
	log.Print("检查配置文件是否正确!")
	cmd := fmt.Sprintf("named-checkzone -i full -q -s full  %s /var/named/zone/%s", zone, remoteFileName)
	_, code := exec_cmd(cmd)
	if code != 0 {
		//log.Fatal(out)
		return false
	}
	return true
}

// checkFIle 检查远端文件是否存在
func checkFIle(remoteFileName string) bool {

	// 判断远程文件是否存在
	cmd := fmt.Sprintf("test -e %s ", remoteFileName)

	out, code := exec_cmd(cmd)
	if code != 0 {
		log.Printf(out)
		return false
	}

	return true
}
