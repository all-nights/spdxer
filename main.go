// This file is part of spdexer.

// Copyright (C) 2022 Ade M Ramdani.
// SPDX-License-Identifier: GPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
package main

import (
	"bytes"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "spdexer",
		Usage: "spdexer is a cli tool to automate adding SPDX licenses to your go project",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "exclude",
				Usage: "exclude paths",
			},
			&cli.StringFlag{
				Name:  "license",
				Usage: "license to add to files e.g GPL30, MIT, GPL30ORLATER etc",
				Value: "GPL30ORLATER",
			},
			&cli.StringFlag{
				Name:     "name",
				Usage:    "name of the project",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "author",
				Usage:    "author of the project",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "year",
				Usage:    "year of the project",
				Required: true,
			},
		},
	}

	app.Action = parse

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

var licenses = map[string]string{
	"GPL30ORLATER": GPL30ORLATER,
}

func findLineStart(src string) int {
	var pkgLine int
	// scan the source line by line
	for lineNum, line := range strings.Split(src, "\n") {
		// if the line is a package declaration
		if strings.HasPrefix(line, "package ") {
			pkgLine = lineNum + 1
			break
		}
	}

	var (
		licenseLine []int
	)

	if pkgLine != 1 {
		for lineNum, line := range strings.Split(src, "\n") {
			if lineNum == pkgLine-1 {
				break
			}

			for _, license := range licenses {
				// find similarity of the license
				if strings.Contains(line, license) {
					licenseLine = append(licenseLine, lineNum+1)
				}
			}
		}
	}

	sort.Ints(licenseLine)

	var lineStart int

	if len(licenseLine) > 0 {
		endLicenseLine := licenseLine[len(licenseLine)-1]
		lineStart = endLicenseLine
	} else {
		lineStart = pkgLine
	}

	return lineStart
}

func parse(ctx *cli.Context) error {
	tmp := licenses[ctx.String("license")]
	excludePaths := ctx.StringSlice("exclude")
	var data TemplateData
	data.Author = ctx.String("author")
	data.Name = ctx.String("name")
	data.Year = ctx.String("year")

	licenseTemplate := template.Must(template.New("").Parse(tmp))

	var licenseText bytes.Buffer
	err := licenseTemplate.Execute(&licenseText, data)
	if err != nil {
		return err
	}

	var paths []string
	err = filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			for _, excludePath := range excludePaths {
				if strings.HasPrefix(path, excludePath) {
					return nil
				}
			}
			abs, _ := filepath.Abs(path)
			paths = append(paths, abs)
		}
		return nil
	})

	for _, path := range paths {
		src, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		lineStart := findLineStart(string(src))

		var newSrc []byte
		newSrc = append(newSrc, licenseText.Bytes()...)
		newSrc = append(newSrc, trimSource(src, lineStart)...)

		err = ioutil.WriteFile(path, newSrc, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func trimSource(src []byte, start int) []byte {
	var newSrc []byte
	for i, line := range strings.Split(string(src), "\n") {
		if i < start-1 {
			continue
		}
		newSrc = append(newSrc, []byte(line+"\n")...)
	}
	return newSrc
}


