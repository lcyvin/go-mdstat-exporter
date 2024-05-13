package main

import (
  "fmt"
  "github.com/lcyvin/go-mdstat-exporter/collector"
)

func main() {
  mdstat, err := collector.NewMdstatData()
  if err != nil {
    fmt.Println(err)
  }

  fmt.Println("Arrays:")
  for _, array := range mdstat.Arrays {
    fmt.Println("Name: "+array.Array)
    fmt.Println("Current Operation: "+string(array.OpStatus.Type))
    fmt.Println("Operation Progress: "+fmt.Sprintf("%f%%", (float64(array.OpStatus.OpProgress)/float64(array.OpStatus.OpTotal))*100))
    bmmc, err := array.BlockMismatchCount()
    if err != nil {
      fmt.Println(err)
    }

    fmt.Println("Block Mismatch Count: "+fmt.Sprintf("%d", bmmc))
  }

}
