package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	fileutil "github.com/projectdiscovery/utils/file"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// table2csv converts a markdown table to csv

var (
	MarkdownInput string // can be a file or a folder
	outputCsv     string // output csv file
	mytable       = &Table{
		Header: []string{},
		Rows: [][]string{
			{},
		},
	}
)

// Table represents a markdown table
type Table struct {
	Header []string
	Rows   [][]string
}

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
	// Read the file and split it
	// Write the file to the output folder
	bin, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	mdx := goldmark.New(goldmark.WithExtensions(extension.Table))
	mdx.Parser()

	mdParser := parser.NewParser(
		parser.WithBlockParsers(parser.DefaultBlockParsers()...),
		parser.WithInlineParsers(parser.DefaultInlineParsers()...),
		parser.WithParagraphTransformers(util.Prioritized(extension.NewTableParagraphTransformer(), 100)),
	)

	rootNode := mdParser.Parse(text.NewReader(bin))

	rowIndex := 0

	ast.Walk(rootNode, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch node.Kind() {
			case extast.KindTableCell:
				if node.Parent().Kind() == extast.KindTableHeader {
					mytable.Header = append(mytable.Header, string(node.Text(bin)))
				} else if node.Parent().Kind() == extast.KindTableRow {
					mytable.Rows[rowIndex] = append(mytable.Rows[rowIndex], string(node.Text(bin)))
				}
			}
		} else {
			switch node.Kind() {

			case extast.KindTableRow:
				rowIndex++
				mytable.Rows = append(mytable.Rows, []string{})
			}
		}
		return ast.WalkContinue, nil
	})
	return nil
}
