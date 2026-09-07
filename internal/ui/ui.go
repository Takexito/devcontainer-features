// Package ui — подтверждения и таблицы для терминала.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// Confirm спрашивает через /dev/tty, чтобы работать и когда stdin занят
// пайпом; без терминала отвечает «нет».
func Confirm(prompt string) bool {
	in := os.Stdin
	if tty, err := os.Open("/dev/tty"); err == nil {
		defer func() { _ = tty.Close() }()
		in = tty
	}
	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)
	line, _ := bufio.NewReader(in).ReadString('\n')
	ans := strings.TrimSpace(line)
	return ans == "y" || ans == "Y"
}

// Table выравнивает колонки по рунам: printf считает байты, и кириллица
// разъезжает колонки.
func Table(w io.Writer, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, r := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(r, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}
