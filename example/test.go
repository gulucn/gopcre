package main

import (
	"bufio"
	_ "bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gulucn/gopcre"
)

func loadData(src *os.File, path string) {
	fmt.Printf("load data from %v\n", path)
	cmdbegin_matcher := pcre.MustCompile("^T:(\\d+)\\((\\d\\d:\\d\\d:\\d\\d)\\).*\\] Got command: (-*\\d+) -(.*) from", 0)
	defer cmdbegin_matcher.Close()
	begin := time.Now()
	reader := bufio.NewReaderSize(src, 16777216)
	cnt := 0
	datasize := 0
	for {
		var line []byte = nil
		buf, err := reader.ReadSlice('\n')
		if err == bufio.ErrBufferFull {
			line = append(line, buf...)
			continue
		} else if line == nil {
			line = buf
		} else {
			line = append(line, buf...)
		}
		if len(line) > 0 || err != io.EOF {
			cnt++
			datasize += len(line)
			if cnt%1024000 == 0 {
				fmt.Printf("hand line:%d,time:%v\n", cnt, time.Now().Sub(begin))
			}
			match, _ := cmdbegin_matcher.Match(line, 0)
			if match != nil {
				//thread := match[1]
				//match_time := match[2]
				//cmd := match[3]
				//uid := match[4]
			}
		}
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("read client data error:%v\n", err)
			break
		}
	}
	fmt.Printf("load data spend: %v,line:%d,data:%d\n", time.Now().Sub(begin), cnt, datasize)
}

func main() {

	if len(os.Args) == 2 {
		path := os.Args[1]
		f, err := os.Open(path)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		loadData(f, path)
	} else {
		loadData(os.Stdin, "stdin")
	}
}
