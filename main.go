package main

import (
	"fmt"
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/os/genv"
	"rsshub-auto-refresh/component"
	"rsshub-auto-refresh/job"
)

func main() {
	configEvn := genv.Get("GF_GCFG_FILE", "")
	if configEvn != "" {
		g.Cfg().SetFileName(configEvn)
	}
	component.InitDatabase()
	component.InitES()
	fmt.Println("Start job")
	job.RegisterJob()
	select {}
}
