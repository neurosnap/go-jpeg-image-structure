package main

import (
	"fmt"
	"os"

	"encoding/json"
	"io/ioutil"

	"github.com/dsoprea/go-exif/v2"
	"github.com/dsoprea/go-jpeg-image-structure"
	"github.com/dsoprea/go-logging"
	"github.com/jessevdk/go-flags"
)

var (
	options = &struct {
		Filepath string `short:"f" long:"filepath" required:"true" description:"File-path of JPEG image ('-' for STDIN)"`
		Json     bool   `short:"j" long:"json" description:"Print as JSON"`
	}{}
)

type SegmentResult struct {
	MarkerId   byte   `json:"marker_id"`
	MarkerName string `json:"marker_name"`
	Offset     int    `json:"offset"`
	Data       []byte `json:"data"`
}

type SegmentIndexItem struct {
	MarkerName string `json:"marker_name"`
	Offset     int    `json:"offset"`
	Data       []byte `json:"data"`
}

func main() {
	defer func() {
		if errRaw := recover(); errRaw != nil {
			err := errRaw.(error)
			log.PrintError(err)

			os.Exit(-2)
		}
	}()

	_, err := flags.Parse(options)
	if err != nil {
		os.Exit(-1)
	}

	var data []byte
	if options.Filepath == "-" {
		var err error
		data, err = ioutil.ReadAll(os.Stdin)
		log.PanicIf(err)
	} else {
		var err error
		data, err = ioutil.ReadFile(options.Filepath)
		log.PanicIf(err)
	}

	jmp := jpegstructure.NewJpegMediaParser()

	intfc, parseErr := jmp.ParseBytes(data)

	var et []exif.ExifTag
	if intfc != nil {
		// If the parse failed, we should always still get all of the segments
		// that we've encountered so far. It should never be empty, and it
		// should be impossible for it to be `nil`. So, if the parse failed but
		// we still found EXIF data, just ignore the failure and proceed. We had
		// still got what we needed.

		sl := intfc.(*jpegstructure.SegmentList)

		var err error
		_, _, et, err = sl.DumpExif()

		// There was a parse error and we couldn't find/parse EXIF data. Panic
		// with the original error from above.
		if err != nil {
			log.Panic(parseErr)
		}
	} else if parseErr == nil {
		// We should never get a `nil` `intfc` value back *and* a `nil`
		// `parseErr`.
		log.Panicf("could not parse JPEG even partially")
	} else {
		log.Panic(parseErr)
	}

	// If we get here, we either parsed the JPEG file well or at least parsed
	// enough to find EXIF data.

	if et == nil {
		// The JPEG image parsed fine (if it didn't and we haven't yet
		// terminated, we already extracted the EXIF tags above).

		sl := intfc.(*jpegstructure.SegmentList)

		var err error

		_, _, et, err = sl.DumpExif()
		if err != nil {
			if err == exif.ErrNoExif {
				fmt.Printf("No EXIF.\n")
				os.Exit(10)
			}

			log.Panic(err)
		}
	}

	if options.Json == true {
		raw, err := json.MarshalIndent(et, "  ", "  ")
		log.PanicIf(err)

		fmt.Println(string(raw))
	} else {
		if len(et) == 0 {
			fmt.Printf("EXIF data is present but empty.\n")
		} else {
			for i, tag := range et {
				fmt.Printf("%2d: IFD-PATH=[%s] ID=(0x%02x) NAME=[%s] TYPE=(%d):[%s] VALUE=[%v]", i, tag.IfdPath, tag.TagId, tag.TagName, tag.TagTypeId, tag.TagTypeName, tag.Value)

				if tag.ChildIfdPath != "" {
					fmt.Printf(" CHILD-IFD-PATH=[%s]", tag.ChildIfdPath)
				}

				fmt.Printf("\n")
			}
		}
	}
}
