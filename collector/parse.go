package collector

import (
	"bufio"
	"errors"
	"os"
  "fmt"
	"strconv"
	"strings"
)

type RaidLevel string

const (
  RaidLevel0  RaidLevel = "raid0"
  RaidLevel1  RaidLevel = "raid1"
  RaidLevel4  RaidLevel = "raid4"
  RaidLevel5  RaidLevel = "raid5"
  RaidLevel6  RaidLevel = "raid6"
  RaidLevel10 RaidLevel = "raid10"
)

type SuperblockType string

const (
  Super09 SuperblockType = "0.9"
  Super1  SuperblockType = "1.0"
  Super11 SuperblockType = "1.1"
  Super12 SuperblockType = "1.2"
)

type OpStatusType string

const (
  OpStatusTypeRecovery OpStatusType = "recovery"
  OpStatusTypeCheck    OpStatusType = "check"
)

type ArrayDevice struct {
  Dev       string  `json:"dev"`
  ArrayIdx  int     `json:"array_index"`
  InUse     bool    `json:"device_in_use"`
  IsFailing bool    `json:"is_failing"`
}

type Bitmap struct {
  PagesTotal int64  `json:"pages_total"`
  PagesUsed  int64  `json:"pages_used"`
}

func ParseBitmapData(data string) (*Bitmap, error) {
  return nil, nil
}

type OpStatus struct {
  Type        OpStatusType  `json:"op_status_type"`
  OpProgress  int64         `json:"op_progress"`
  OpTotal     int64         `json:"op_total"`
}

func ParseOpStatus(data string) (*OpStatus, error) {
  return nil, nil
}

func (ops *OpStatus) ProgressPercent() float32 {
  return (float32(ops.OpProgress)/float32(ops.OpTotal))*100
}

type MdstatData struct {
  Personalities   []RaidLevel `json:"personalities"`
  Arrays          []*ArrayData `json:"arrays"`
}

type ArrayData struct {
  Array           string          `json:"array"`
  State           string          `json:"state"`
  Level           RaidLevel       `json:"raid_level"`
  Disks           []*ArrayDevice   `json:"disks"`
  Blocks          int64           `json:"blocks"`
  SuperblockType  SuperblockType  `json:"superblock_type"`
  ChunkSize       int64           `json:"chunk_size"`
  BitmapStatus    *Bitmap          `json:"bitmap"`
  OpStatus        *OpStatus        `json:"op_status"`
}

func (ad *ArrayData) Device(devName string) (*ArrayDevice, error) {
  for _, dev := range ad.Disks {
    if dev.Dev == devName {
      return dev, nil
    }
  }

  return nil, errors.New("No match found for device name "+devName)
}

func (ad *ArrayData) DeviceByIdx(idx int) (*ArrayDevice, error) {
  for _, dev := range ad.Disks {
    if dev.ArrayIdx == idx {
      return dev, nil
    }
  }

  return nil, errors.New("No match found for device idx "+fmt.Sprintf("%d", idx))
}

func ParseArrayData(arrayData []string) (*ArrayData, error) {
  ad := &ArrayData{
    Disks: []*ArrayDevice{},
  }

  firstLine := arrayData[0]
  arrayNameInfo := strings.Split(firstLine, ":")
  if len(arrayNameInfo) < 2 {
    return nil, errors.New("Unable to parse first line of array data")
  }

  ad.Array = strings.TrimSpace(arrayNameInfo[0])
  arrayInfo := strings.Split(strings.TrimSpace(arrayNameInfo[1]), " ")

  ad.State = arrayInfo[0]
  ad.Level = RaidLevel(arrayInfo[1])

  for _, disk := range arrayInfo[2:] {
    dev := &ArrayDevice{IsFailing: false}
    devName, idx, _ := strings.Cut(disk, "[")
    dev.Dev = devName
    if strings.Index(idx, "(F)") > -1 {
      dev.IsFailing = true
      idx = strings.TrimSuffix(idx, "(F)")
    }

    arrayIdx, err := strconv.Atoi(strings.TrimSuffix(idx, "]"))
    if err != nil {
      return nil, err
    }

    dev.ArrayIdx = arrayIdx
    ad.Disks = append(ad.Disks, dev)
  }

  secondLine := strings.Split(arrayData[1], ",")
  blockInfo := secondLine[0]
  blocks := strings.Split(blockInfo, " ")[0]
  superBlockType := SuperblockType(strings.Split(blockInfo, " ")[3])

  parseBlockstoInt, err := strconv.ParseInt(blocks, 10, 64)
  if err != nil {
    return nil, err
  }

  ad.Blocks = parseBlockstoInt
  ad.SuperblockType = superBlockType

  if len(arrayData) > 2 {
    for _, line := range arrayData[2:] {
      if strings.Contains(line, "bitmap:") {
        bitmap, err := ParseBitmapData(line)
        if err != nil {
          continue
        }

        ad.BitmapStatus = bitmap
      }

      if strings.Contains(line, "check") || strings.Contains(line, "recovery") {
        opStatus, err := ParseOpStatus(line)
        if err != nil {
          continue
        }

        ad.OpStatus = opStatus
      }
    }
  }

  return ad, nil
}

func NewMdstatData() (*MdstatData, error) {
  msd := &MdstatData{
    Personalities: []RaidLevel{},
  }

  f, err := os.Open("/proc/mdstat")
  if err != nil {
    return nil, err
  }

  scanner := bufio.NewScanner(f)
  scanner.Scan()

  if scanner.Err() != nil{
    return nil, err
  }

  personalities, _ := strings.CutPrefix("personalities : ", scanner.Text())
  for _, raidLvl := range strings.Split(strings.TrimSpace(personalities), " ") {
    msd.Personalities = append(msd.Personalities, RaidLevel(strings.Trim(raidLvl, "[]")))
  }
    
  arrays := []*ArrayData{}

  arrayDataLines := make([][]string, 0)
  idx := -1
  for scanner.Scan() {
    if scanner.Err() != nil {
      return nil, err
    }

    line := strings.TrimSpace(scanner.Text())
    // check if we have reached the "unused devices" line
    unusedIdx := strings.Index(line, "unused devices:")
    if unusedIdx > -1 {
      break // not using this right now because I can't find the output format online, it seems to generaly show "<none>"
    }

    // check if this is a new array or not
    mdIdx := strings.Index(line, "md")
    sepIdx := strings.Index(line, " : ")
    if mdIdx > -1 && sepIdx > mdIdx {
      idx += 1
      arrayDataLines = append(arrayDataLines, make([]string, 0))
    }

    arrayDataLines[idx] = append(arrayDataLines[idx], strings.TrimSpace(line))
  }

  for _, rawArrayData := range arrayDataLines {
    arrayData, err := ParseArrayData(rawArrayData)
    if err != nil {
      return nil, err
    }

    arrays = append(arrays, arrayData)
  }

  msd.Arrays = arrays

  return msd, nil
}

func (ad *ArrayData) BlockMismatchCount() (int64, error) {
  f, err := os.Open("/sys/block/"+ad.Array+"/md/mismatch_cnt")
  if err != nil {
    return 0, err
  }
  
  var mismatchCnt []byte
  _, err = f.Read(mismatchCnt)
  if err != nil {
    return 0, err
  }

  out, err := strconv.Atoi(string(mismatchCnt))
  if err != nil {
    return 0, err
  }

  return int64(out), nil
}
