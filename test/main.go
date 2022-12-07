package main

import (
	"fmt"
	"ormtest/zdns"
)

//// 执行bash命令，返回输出以及执行后退出状态码
//func Bash(cmd string) (out string, exitcode int) {
//	cmdobj := exec.Command("bash", "-c", cmd)
//	output, err := cmdobj.CombinedOutput()
//	if err != nil {
//		// Get the exitcode of the output
//		if ins, ok := err.(*exec.ExitError); ok {
//			out = string(output)
//			exitcode = ins.ExitCode()
//			return out, exitcode
//		}
//	}
//	return string(output), 0
//}

func main() {

	server := zdns.Server{
		Host:    "10.200.192.13",
		User:    "root",
		Pwd:     "pSEqXW5AOyJReBVY",
		Port:    22,
		ZoneDir: "/var/named/zone",
	}

	z := zdns.NewZdns(&server)

	client, err := z.Connect()
	fmt.Println(client)
	if err != nil {
		return
	}

	z.Init(client)

	record := zdns.Record{
		Name: "test",
		Type: "A",
		Ttl:  60,
		Addr: "192.168.100.21",
	}

	var records []*zdns.Record
	records = append(records, &record)

	domain := zdns.Domain{
		Records:   records,
		Zone:      "abc.com",
		Operation: "del",
	}

	if res := z.DomainOperation(&domain); res {
		fmt.Print(res)
		fmt.Printf("Success!! ")
	} else {
		fmt.Printf("Faild")
	}

	//zone := &zdns.Zone{
	//	Name:      "abc.com",
	//	Server:    "10.200.192.13",
	//	Operation: "status",
	//}
	//
	//if out, res := z.Zone(zone); res {
	//	fmt.Printf(out)
	//	fmt.Printf("Success!! ")
	//} else {
	//	fmt.Printf("Faild")
	//}

	//out, res := z.ZoneStatus()
	//if res {
	//	fmt.Println(out)
	//}

	//z.ZoneCreate()

	//domain := zdns.Domain{
	//	Name:   "test",
	//	Record: "A",
	//	Ttl:    60,
	//	Addr:   "192.168.100.21",
	//}
	//
	//var domains []*zdns.Domain
	//domains = append(domains, &domain)
	//
	//zdns := zdns.Zdomain{
	//	domains,
	//	"add",
	//}
	//
	//z.DomainAdd(&zdns)

}
