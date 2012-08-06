package cablastp

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type DBConf struct {
	MinMatchLen         int
	MatchKmerSize       int
	GappedWindowSize    int
	UngappedWindowSize  int
	ExtSeqIdThreshold   int
	MatchSeqIdThreshold int
	MatchExtend         int
	MapSeedSize         int
	ExtSeedSize         int
	SavePlain           bool
}

var DefaultDBConf = DBConf{
	MinMatchLen:         40,
	MatchKmerSize:       4,
	GappedWindowSize:    25,
	UngappedWindowSize:  10,
	ExtSeqIdThreshold:   50,
	MatchSeqIdThreshold: 60,
	MatchExtend:         30,
	MapSeedSize:         6,
	ExtSeedSize:         4,
	SavePlain:           false,
}

func LoadDBConf(r io.Reader) (conf DBConf, err error) {
	defer func() {
		if perr := recover(); perr != nil {
			err = perr.(error)
		}
	}()
	conf = DefaultDBConf
	csvReader := csv.NewReader(r)
	csvReader.Comma = ':'
	csvReader.Comment = '#'
	csvReader.FieldsPerRecord = 2
	csvReader.TrailingComma = false
	csvReader.TrimLeadingSpace = true

	lines, err := csvReader.ReadAll()
	if err != nil {
		return conf, err
	}

	for _, line := range lines {
		atoi := func() int {
			var i64 int64
			var err error
			if i64, err = strconv.ParseInt(line[1], 10, 32); err != nil {
				panic(err)
			}
			return int(i64)
		}
		switch line[0] {
		case "MinMatchLen":
			conf.MinMatchLen = atoi()
		case "MatchKmerSize":
			conf.MatchKmerSize = atoi()
		case "GappedWindowSize":
			conf.GappedWindowSize = atoi()
		case "UngappedWindowSize":
			conf.UngappedWindowSize = atoi()
		case "ExtSeqIdThreshold":
			conf.ExtSeqIdThreshold = atoi()
		case "MatchSeqIdThreshold":
			conf.MatchSeqIdThreshold = atoi()
		case "MatchExtend":
			conf.MatchExtend = atoi()
		case "MapSeedSize":
			conf.MapSeedSize = atoi()
		case "ExtSeedSize":
			conf.ExtSeedSize = atoi()
		case "SavePlain":
			if strings.TrimSpace(line[1]) == "1" {
				conf.SavePlain = true
			} else {
				conf.SavePlain = false
			}
		default:
			return conf, fmt.Errorf("Invalid DBConf flag: %s", line[0])
		}
	}

	return conf, nil
}

func (flagConf DBConf) FlagMerge(fileConf DBConf) (DBConf, error) {
	only := make(map[string]bool, 0)
	flag.Visit(func(f *flag.Flag) { only[f.Name] = true })

	if only["map-seed-size"] {
		return flagConf, fmt.Errorf("The map seed size cannot be changed for " +
			"an existing database.")
	}

	if !only["min-match-len"] {
		flagConf.MinMatchLen = fileConf.MinMatchLen
	}
	if !only["match-kmer-size"] {
		flagConf.MatchKmerSize = fileConf.MatchKmerSize
	}
	if !only["gapped-window-size"] {
		flagConf.GappedWindowSize = fileConf.GappedWindowSize
	}
	if !only["ungapped-window-size"] {
		flagConf.UngappedWindowSize = fileConf.UngappedWindowSize
	}
	if !only["ext-seq-id-threshold"] {
		flagConf.ExtSeqIdThreshold = fileConf.ExtSeqIdThreshold
	}
	if !only["match-seq-id-threshold"] {
		flagConf.MatchSeqIdThreshold = fileConf.MatchSeqIdThreshold
	}
	if !only["match-extend"] {
		flagConf.MatchExtend = fileConf.MatchExtend
	}
	if !only["ext-seed-size"] {
		flagConf.ExtSeedSize = fileConf.ExtSeedSize
	}
	if !only["plain"] {
		flagConf.SavePlain = fileConf.SavePlain
	}
	return flagConf, nil
}

func (dbConf DBConf) Write(w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	csvWriter.Comma = ':'
	csvWriter.UseCRLF = false

	s := func(i int) string {
		return fmt.Sprintf("%d", i)
	}
	bs := func(b bool) string {
		if b {
			return "1"
		}
		return "0"
	}
	records := [][]string{
		{"MinMatchLen", s(dbConf.MinMatchLen)},
		{"MatchKmerSize", s(dbConf.MatchKmerSize)},
		{"GappedWindowSize", s(dbConf.GappedWindowSize)},
		{"UngappedWindowSize", s(dbConf.UngappedWindowSize)},
		{"ExtSeqIdThreshold", s(dbConf.ExtSeqIdThreshold)},
		{"MatchSeqIdThreshold", s(dbConf.MatchSeqIdThreshold)},
		{"MatchExtend", s(dbConf.MatchExtend)},
		{"MapSeedSize", s(dbConf.MapSeedSize)},
		{"ExtSeedSize", s(dbConf.ExtSeedSize)},
		{"SavePlain", bs(dbConf.SavePlain)},
	}
	if err := csvWriter.WriteAll(records); err != nil {
		return err
	}
	return nil
}