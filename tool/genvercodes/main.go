package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/filecoin-project/lotus/chain/actors"
)

const (
	separatedTemplateExtName = ".sep.template"
	combinedTemplateExtName  = ".one.template"
)

type verInfo struct {
	Ver        int
	ImportPath string
}

type genOption struct {
	ext   string
	genfn func(gt *genTemplate, vers []verInfo) error
}

type dirOption struct {
	In  string
	Out string
}

func main() {
	dirs := []dirOption{
		{
			In:  "./racailum/segment/extract/actorstate.templates/",
			Out: "./racailum/segment/extract/actorstate/",
		},
		{
			In:  "./racailum/segment/actor.templates/",
			Out: "./racailum/segment/actor/",
		},
	}

	opts := []genOption{
		{
			ext:   separatedTemplateExtName,
			genfn: genSeparatedForFile,
		},
		{
			ext:   combinedTemplateExtName,
			genfn: genCombinedForFile,
		},
	}

	vers := make([]verInfo, len(actors.Versions))
	for i := range actors.Versions {
		ver := actors.Versions[i]
		inPath := "github.com/filecoin-project/specs-actors"
		if ver != 0 {
			inPath = fmt.Sprintf("%s/v%d", inPath, ver)
		}

		vers[i] = verInfo{
			Ver:        ver,
			ImportPath: inPath,
		}
	}

	for _, dir := range dirs {
		for _, opt := range opts {
			pattern := filepath.Join(dir.In, "*"+opt.ext)
			fpaths, err := filepath.Glob(pattern)
			if err != nil {
				log.Fatalf("glob for pattern %s: %s", pattern, err)
			}

			for _, fpath := range fpaths {
				gt, err := parseGenTemplate(fpath, opt.ext, dir.Out)
				if err != nil {
					log.Fatalf("parse generating template %s: %s", fpath, err)
				}

				err = opt.genfn(gt, vers)
				if err != nil {
					log.Fatalf("generate for template %s: %s", gt.basename, err)
				}
			}
		}

	}
}

type genTemplate struct {
	inDir    string
	basename string
	filename string
	outDir   string
	vers     verRanges
	t        *template.Template
}

func parseGenTemplate(fpath string, ext string, outDir string) (*genTemplate, error) {
	dir := filepath.Dir(fpath)
	baseName := filepath.Base(fpath)
	fileName := baseName[:len(baseName)-len(ext)]

	data, err := os.ReadFile(fpath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	verDefStr := ""
	firstNL := bytes.IndexByte(data, '\n')
	if firstNL != -1 {
		firstLine := data[:firstNL]
		if bytes.HasPrefix(firstLine, []byte("ver:")) {
			verDefStr = string(bytes.TrimSpace(data[4:firstNL]))
			data = data[firstNL+1:]
		}
	}

	var vrs verRanges
	if verDefStr != "" {
		rangePieces := strings.Split(verDefStr, ";")
		for _, piece := range rangePieces {
			if content := strings.TrimSpace(piece); content != "" {
				vr, err := parseVerRange(content)
				if err != nil {
					return nil, fmt.Errorf("parse ver range string %s: %w", piece, err)
				}

				vrs = append(vrs, vr)
			}
		}
	}

	t := template.New(fileName)
	t, err = t.Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse template text: %w", err)
	}

	return &genTemplate{
		inDir:    dir,
		basename: baseName,
		filename: fileName,
		outDir:   outDir,
		vers:     vrs,
		t:        t,
	}, nil
}

func genCombinedForFile(gt *genTemplate, vers []verInfo) error {
	log.Printf("generating for combined template %s", gt.basename)
	defer log.Println("")

	filtered := make([]verInfo, 0, len(vers))
	for vi := range vers {
		if gt.vers.match(vers[vi].Ver) {
			filtered = append(filtered, vers[vi])
		}
	}

	if len(filtered) == 0 {
		log.Println("\tno code generated")
		return nil
	}

	var outbuf bytes.Buffer
	err := gt.t.Execute(&outbuf, map[string]interface{}{
		"Vers":  filtered,
		"First": filtered[0].Ver,
		"Last":  filtered[len(filtered)-1].Ver,
	})

	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	formatted, err := format.Source(outbuf.Bytes())
	if err != nil {
		return fmt.Errorf("format go source file for rendered file: %w", err)
	}

	outFileName := fmt.Sprintf("%s.go", gt.filename)
	outFilePath := filepath.Join(gt.outDir, outFileName)
	err = os.WriteFile(outFilePath, formatted, 0644)
	if err != nil {
		return fmt.Errorf("write into out file %s: %w", outFilePath, err)
	}

	log.Printf("\tgenerated combined file to %s", outFilePath)

	return nil
}

func genSeparatedForFile(gt *genTemplate, vers []verInfo) error {
	log.Printf("generating for separated template %s", gt.basename)

	var outbuf bytes.Buffer
	for _, vi := range vers {
		if !gt.vers.match(vi.Ver) {
			continue
		}

		outbuf.Reset()
		err := gt.t.Execute(&outbuf, vi)
		if err != nil {
			return fmt.Errorf("render template for ver %d: %w", vi.Ver, err)
		}

		formatted, err := format.Source(outbuf.Bytes())
		if err != nil {
			return fmt.Errorf("format go source file for rendered file of ver %d: %w", vi.Ver, err)
		}

		outFileName := fmt.Sprintf("%s_v%d.go", gt.filename, vi.Ver)
		outFilePath := filepath.Join(gt.outDir, outFileName)
		err = os.WriteFile(outFilePath, formatted, 0644)
		if err != nil {
			return fmt.Errorf("write into out file %s: %w", outFilePath, err)
		}

		log.Printf("\tgenerated ver %d to %s", vi.Ver, outFilePath)
	}
	log.Println("")

	return nil
}

type verRanges []*verRange

func (vrs verRanges) match(expect int) bool {
	if len(vrs) == 0 {
		return true
	}

	for _, vr := range vrs {
		if vr.match(expect) {
			return true
		}
	}

	return false
}

func parseVerRange(s string) (*verRange, error) {
	pieces := strings.SplitN(s, "-", 2)
	if len(pieces) == 1 {
		ver, err := strconv.Atoi(strings.TrimSpace(pieces[0]))
		if err != nil {
			return nil, fmt.Errorf("convert %s to number: %w", pieces[0], err)
		}

		return &verRange{
			min: &ver,
			max: &ver,
		}, nil
	}

	var min, max *int
	if left := strings.TrimSpace(pieces[0]); left != "" {
		minv, err := strconv.Atoi(left)
		if err != nil {
			return nil, fmt.Errorf("convert min value %s to number: %w", left, err)
		}

		min = &minv
	}

	if right := strings.TrimSpace(pieces[1]); right != "" {
		maxv, err := strconv.Atoi(right)
		if err != nil {
			return nil, fmt.Errorf("convert max value %s to number: %w", right, err)
		}

		max = &maxv
	}

	return &verRange{
		min: min,
		max: max,
	}, nil
}

type verRange struct {
	min *int
	max *int
}

func (vr *verRange) match(expect int) bool {
	if vr.min != nil && expect < *vr.min {
		return false
	}

	if vr.max != nil && expect > *vr.max {
		return false
	}

	return true
}
