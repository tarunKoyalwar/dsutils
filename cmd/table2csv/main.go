package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	fileutil "github.com/projectdiscovery/utils/file"
	"github.com/tarunKoyalwar/dsutils/pkg/table2csv"
)

// table2csv converts a markdown table to csv

var (
	MarkdownInput string // can be a file or a folder
	outputCsv     string // output csv file
	mytable       = &table2csv.Table{
		Header: []string{},
		Rows: [][]string{
			{},
		},
	}
)

func main() {
	flag.StringVar(&MarkdownInput, "input", "", "Markdown file or folder to split")
	flag.StringVar(&outputCsv, "output", "", "Output csv file")
	flag.Parse()

	if MarkdownInput == "" || !fileutil.FileOrFolderExists(MarkdownInput) || outputCsv == "" {
		// Split all markdown files in the folder
		flag.Usage()
		return
	}

	switch {
	case fileutil.FileExists(MarkdownInput):
		fmt.Printf("[+] Processing %v\n", MarkdownInput)
		processFileCallback(MarkdownInput)
	case fileutil.FolderExists(MarkdownInput):
		filepath.WalkDir(MarkdownInput, func(path string, d os.DirEntry, err error) error {
			if d.IsDir() || filepath.Ext(path) != ".md" || err != nil {
				return nil
			}
			fmt.Printf("[+] Processing %v\n", path)
			err = processFileCallback(path)
			if err != nil {
				fmt.Printf("[-] Error processing %v: %v\n", path, err)
			}
			return nil
		})
	}

	file, err := os.Create(outputCsv)
	if err != nil {
		log.Fatalf("[-] Error creating output file: %v\n", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write(mytable.Header)
	writer.WriteAll(mytable.Rows)

}

func processFileCallback(filename string) error {
	return table2csv.ParseMarkdownIntoTable(filename, mytable)
}
