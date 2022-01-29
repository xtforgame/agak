package main

import (
	// "fmt"
	"github.com/xtforgame/agak/mainhelpers"
	"github.com/xtforgame/agak/utils"
)

func main() {
	defer utils.FinalReport()
	ms := mainhelpers.NewSbMainServiceForDev()
	ms.Init()
	ms.Start()
	defer ms.Destroy()
	// os.Exit(0)
}
