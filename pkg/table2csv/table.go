package table2csv

import (
	"math"
	"os"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Table represents a markdown table
type Table struct {
	Header []string
	Rows   [][]string
}

// ParseMarkdownIntoTable parses a markdown file into a table
func ParseMarkdownIntoTable(filename string, mytable *Table) error {
	// Read the file and split it
	// Write the file to the output folder
	bin, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	if mytable == nil {
		mytable = &Table{
			Header: []string{},
			Rows: [][]string{
				{},
			},
		}
	}

	mdx := goldmark.New(goldmark.WithExtensions(extension.Table))
	mdx.Parser()

	mdParser := parser.NewParser(
		parser.WithBlockParsers(parser.DefaultBlockParsers()...),
		parser.WithInlineParsers(parser.DefaultInlineParsers()...),
		parser.WithParagraphTransformers(util.Prioritized(extension.NewTableParagraphTransformer(), 100)),
	)

	rootNode := mdParser.Parse(text.NewReader(bin))

	if mytable.Rows == nil {
		mytable.Rows = [][]string{
			{},
		}
	}

	rowIndex := len(mytable.Rows) - 1
	rowIndex = int(math.Max(0, float64(rowIndex)))

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

	if len(mytable.Rows) > 1 && len(mytable.Rows[len(mytable.Rows)-1]) == 0 {
		mytable.Rows = mytable.Rows[:len(mytable.Rows)-1]
	}

	return nil
}

// GetTableFromMarkdown returns a table from a markdown file
func GetTableFromMarkdown(filename string) (*Table, error) {
	mytable := &Table{
		Header: []string{},
		Rows: [][]string{
			{},
		},
	}
	err := ParseMarkdownIntoTable(filename, mytable)
	return mytable, err
}
