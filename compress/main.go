package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/BurntSushi/cablastp"
)

var (
	flagGoMaxProcs         = runtime.NumCPU()
	flagMinMatchLen        = 25
	flagMatchKmerSize      = 3
	flagGappedWindowSize   = 25
	flagUngappedWindowSize = 10
	flagSeqIdThreshold     = 50
	flagSeedSize           = 3

	flagCpuProfile = ""
	flagMemProfile = ""
)

func init() {
	log.SetFlags(0)

	flag.IntVar(&flagGoMaxProcs, "p", flagGoMaxProcs,
		"The maximum number of CPUs that can be executing simultaneously.")
	flag.IntVar(&flagMinMatchLen, "min-match-len", flagMinMatchLen,
		"The minimum size of a match.")
	flag.IntVar(&flagMatchKmerSize, "match-kmer-size", flagMatchKmerSize,
		"The size of kmer fragments to match in ungapped extension.")
	flag.IntVar(&flagGappedWindowSize, "gapped-window-size",
		flagGappedWindowSize, "The size of the gapped match window.")
	flag.IntVar(&flagUngappedWindowSize, "ungapped-window-size",
		flagUngappedWindowSize, "The size of the ungapped match window.")
	flag.IntVar(&flagSeqIdThreshold, "seq-id-threshold", flagSeqIdThreshold,
		"The sequence identity threshold of a match. (An integer in the "+
			"inclusive range from 0 to 100.)")
	flag.IntVar(&flagSeedSize, "seed-size", flagSeedSize,
		"The size of a seed.")

	flag.StringVar(&flagCpuProfile, "cpuprofile", flagCpuProfile,
		"When set, a CPU profile will be written to the file specified.")
	flag.StringVar(&flagMemProfile, "memprofile", flagMemProfile,
		"When set, a memory profile will be written to the file specified.")
}

func main() {
	if flag.NArg() < 2 {
		flag.Usage()
	}

	if len(flagCpuProfile) > 0 {
		f, err := os.Create(flagCpuProfile)
		if err != nil {
			fatalf("%s\n", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	orgSeqId := 0
	DB, err := cablastp.NewDB(flag.Arg(0), flagSeedSize, false, true)
	if err != nil {
		fatalf("%s\n", err)
	}

	// Create the compression workers.
	wg := &sync.WaitGroup{}
	jobs := make(chan compressJob, 200)
	for i := 0; i < max(1, runtime.GOMAXPROCS(0)-3); i++ {
		wg.Add(1)
		go compressWorker(DB, jobs, wg)
	}

	start := time.Now()
	for _, arg := range flag.Args()[1:] {
		seqChan, err := cablastp.ReadOriginalSeqs(arg)
		if err != nil {
			log.Fatal(err)
		}
		for readSeq := range seqChan {
			if readSeq.Err != nil {
				log.Fatal(err)
			}

			jobs <- compressJob{
				orgSeqId: orgSeqId,
				orgSeq:   readSeq.Seq,
			}
			orgSeqId++

			if orgSeqId%100 == 0 {
				secElapsed := time.Since(start).Seconds()
				seqsPerSec := float64(orgSeqId) / float64(secElapsed)
				fmt.Printf("\r%d sequences compressed (%0.4f seqs/sec)",
					orgSeqId, seqsPerSec)
			}
		}
	}
	close(jobs)
	wg.Wait()

	if len(flagMemProfile) > 0 {
		writeMemProfile(fmt.Sprintf("%s.%d",
			flagMemProfile, orgSeqId))
	}

	if err := DB.CoarseDB.Save(); err != nil {
		fatalf("Could not save coarse database: %s\n", err)
	}

	DB.Close()

	fmt.Println("\r")
}

func errorf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
}

func fatalf(format string, v ...interface{}) {
	errorf(format, v...)
	os.Exit(1)
}

func writeMemProfile(name string) {
	f, err := os.Create(name)
	if err != nil {
		fatalf("%s\n", err)
	}
	pprof.WriteHeapProfile(f)
	f.Close()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func init() {
	flag.Usage = usage
	flag.Parse()

	runtime.GOMAXPROCS(flagGoMaxProcs)
}

func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: %s [flags] "+
			"database-directory "+
			"fasta-file [fasta-file ...]\n",
		path.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(1)
}
