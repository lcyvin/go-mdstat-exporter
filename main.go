package main

import (
	"fmt"
	"os"

	"github.com/lcyvin/ansifmt"
	"github.com/lcyvin/go-mdstat-exporter/collector"
)

func main() {
  mdstat, err := collector.NewMdstatData()
  if err != nil {
    fmt.Println(err)
  }

  ansifmt.Wrapln("Arrays:", ansifmt.BOLD, ansifmt.UNDERLINE, ansifmt.WHITE)

  for _, array := range mdstat.Arrays {
    ansifmt.NewFormatter().Set(ansifmt.BOLD, ansifmt.UNDERLINE).Append("• %s:").Unset(ansifmt.BOLD).Append("%s\n").Printf("Name", array.Array)
    fmt.Fprintf(os.Stdout, "  \033[1;37m  Operation:\033[0m%12s\n", array.OpStatus.Type)
    if array.OpStatus.Type != collector.OpStatusTypeIdle {
      fmt.Fprintf(os.Stdout, "  \033[1;37m  Progress:\033[0m%12.2f%s\n", array.OpStatus.ProgressPercent(), "%")
      fmt.Fprintf(os.Stdout, "  \033[1;37m  Current Block:\033[0m%13d\n", array.OpStatus.OpProgress)
      fmt.Fprintf(os.Stdout, "  \033[1;37m  Total Blocks:\033[0m%15d\n", array.OpStatus.OpTotal)
      fmt.Fprintf(os.Stdout, "  \033[1;37m  Speed:\033[0m%22s\n", array.OpStatus.Speed)
    }
  }
}
