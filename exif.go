package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/h2non/bimg"
)

type EXIF struct {
	Make                    string   `json:"make,omitempty"`
	Model                   string   `json:"model,omitempty"`
	Orientation             int      `json:"orientation,omitempty"`
	Software                string   `json:"software,omitempty"`
	YCbCrPositioning        int      `json:"ycbcrPositioning,omitempty"`
	ExifVersion             string   `json:"exifVersion,omitempty"`
	ISO                     int      `json:"iso,omitempty"`
	ComponentsConfiguration string   `json:"componentsConfiguration,omitempty"`
	FocalLengthIn35mmFilm   int      `json:"focalLengthIn35mmFilm,omitempty"`
	ExifImageWidth          int      `json:"exifImageWidth,omitempty"`
	ExifImageHeight         int      `json:"exifImageHeight,omitempty"`
	XResolution             string   `json:"xResolution,omitempty"`
	YResolution             string   `json:"yResolution,omitempty"`
	DateTime                string   `json:"dateTime,omitempty"`
	DateTimeOriginal        string   `json:"dateTimeOriginal,omitempty"`
	DateTimeDigitized       string   `json:"dateTimeDigitized,omitempty"`
	FNumber                 string   `json:"fNumber,omitempty"`
	ExposureTime            string   `json:"exposureTime,omitempty"`
	ExposureProgram         any      `json:"exposureProgram,omitempty"`
	ShutterSpeedValue       string   `json:"shutterSpeedValue,omitempty"`
	ApertureValue           string   `json:"apertureValue,omitempty"`
	BrightnessValue         string   `json:"brightnessValue,omitempty"`
	ExposureCompensation    string   `json:"exposureCompensation,omitempty"`
	MeteringMode            any      `json:"meteringMode,omitempty"`
	Compression             any      `json:"compression,omitempty"`
	Flash                   bool     `json:"flash,omitempty"`
	FlashMode               any      `json:"flashMode,omitempty"`
	FocalLength             string   `json:"focalLength,omitempty"`
	SubjectArea             []int    `json:"subjectArea,omitempty"`
	ColorSpace              any      `json:"colorSpace,omitempty"`
	SensingMethod           any      `json:"sensingMethod,omitempty"`
	ExposureMode            any      `json:"exposureMode,omitempty"`
	WhiteBalance            any      `json:"whiteBalance,omitempty"`
	SceneType               string   `json:"sceneType,omitempty"`
	SceneCaptureType        int      `json:"sceneCaptureType,omitempty"`
	GPS                     *EXIFGPS `json:"gps,omitempty"`
	// MakerNote           string `json:"makerNote,omitempty"`
	// SubSecTimeOriginal  string `json:"subSecTimeOriginal,omitempty"`
	// SubSecTimeDigitized string `json:"subSecTimeDigitized,omitempty"`
}

type EXIFGPS struct {
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Altitude     string  `json:"altitude"`
	Speed        string  `json:"speed"`
	Direction    float64 `json:"direction"`
	DirectionRef string  `json:"directionRef"`
}

// ParseEXIFFromBimg returns an EXIF struct by parsing the one returned by bimg
func ParseEXIFFromBimg(s *bimg.EXIF) *EXIF {
	res := &EXIF{
		// Copied as-is
		Make:                    s.Make,
		Model:                   s.Model,
		Orientation:             s.Orientation,
		Software:                s.Software,
		YCbCrPositioning:        s.YCbCrPositioning,
		ExifVersion:             s.ExifVersion,
		ISO:                     s.ISOSpeedRatings,
		ComponentsConfiguration: s.ComponentsConfiguration,
		FocalLengthIn35mmFilm:   s.FocalLengthIn35mmFilm,
		ExifImageWidth:          s.PixelXDimension,
		ExifImageHeight:         s.PixelYDimension,
		SceneType:               s.SceneType,
		SceneCaptureType:        s.SceneCaptureType,
	}

	if s.XResolution != "" {
		res.XResolution = formatResolution(s.XResolution, s.ResolutionUnit)
	}
	if s.YResolution != "" {
		res.YResolution = formatResolution(s.YResolution, s.ResolutionUnit)
	}
	if s.Datetime != "" {
		res.DateTime = formatDateTime(s.Datetime)
	}
	if s.DateTimeOriginal != "" {
		res.DateTimeOriginal = formatDateTime(s.DateTimeOriginal)
	}
	if s.DateTimeDigitized != "" {
		res.DateTimeDigitized = formatDateTime(s.DateTimeDigitized)
	}
	if s.FNumber != "" {
		res.FNumber = formatFloat(parseEXIFRationalOrLogError(s.FNumber), 3)
	}
	if s.ExposureTime != "" {
		res.ExposureTime = formatHumanRational(s.ExposureTime)
	}
	if s.ShutterSpeedValue != "" {
		res.ShutterSpeedValue = formatHumanRational(s.ShutterSpeedValue)
	}
	if s.ApertureValue != "" {
		res.ApertureValue = formatHumanRational(s.ApertureValue)
	}
	if s.BrightnessValue != "" {
		res.BrightnessValue = formatHumanRational(s.BrightnessValue)
	}
	if s.ExposureBiasValue != "" && s.ExposureBiasValue != "0" {
		res.ExposureCompensation = formatHumanRational(s.ExposureBiasValue)
	}
	if s.FocalLength != "" {
		res.FocalLength = formatFloat(parseEXIFRationalOrLogError(s.FocalLength), 2)
	}
	if s.SubjectArea != "" {
		parts := strings.Split(s.SubjectArea, " ")
		if len(parts) >= 2 && len(parts) <= 4 {
			res.SubjectArea = make([]int, len(parts))
			i := 0
			for _, part := range parts {
				n, err := strconv.Atoi(part)
				if err != nil {
					continue
				}
				res.SubjectArea[i] = n
				i++
			}
			res.SubjectArea = res.SubjectArea[:i]
		}
	}
	{
		res.Flash = (s.Flash & 1) == 1
		switch s.Flash {
		case 0x0:
			res.FlashMode = "No Flash"
		case 0x1:
			res.FlashMode = "Fired"
		case 0x5:
			res.FlashMode = "Fired, Return not detected"
		case 0x7:
			res.FlashMode = "Fired, Return detected"
		case 0x8:
			res.FlashMode = "On, Did not fire"
		case 0x9:
			res.FlashMode = "On, Fired"
		case 0xd:
			res.FlashMode = "On, Return not detected"
		case 0xf:
			res.FlashMode = "On, Return detected"
		case 0x10:
			res.FlashMode = "Off, Did not fire"
		case 0x14:
			res.FlashMode = "Off, Did not fire, Return not detected"
		case 0x18:
			res.FlashMode = "Auto, Did not fire"
		case 0x19:
			res.FlashMode = "Auto, Fired"
		case 0x1d:
			res.FlashMode = "Auto, Fired, Return not detected"
		case 0x1f:
			res.FlashMode = "Auto, Fired, Return detected"
		case 0x20:
			res.FlashMode = "No flash function"
		case 0x30:
			res.FlashMode = "Off, No flash function"
		case 0x41:
			res.FlashMode = "Fired, Red-eye reduction"
		case 0x45:
			res.FlashMode = "Fired, Red-eye reduction, Return not detected"
		case 0x47:
			res.FlashMode = "Fired, Red-eye reduction, Return detected"
		case 0x49:
			res.FlashMode = "On, Red-eye reduction"
		case 0x4d:
			res.FlashMode = "On, Red-eye reduction, Return not detected"
		case 0x4f:
			res.FlashMode = "On, Red-eye reduction, Return detected"
		case 0x50:
			res.FlashMode = "Off, Red-eye reduction"
		case 0x58:
			res.FlashMode = "Auto, Did not fire, Red-eye reduction"
		case 0x59:
			res.FlashMode = "Auto, Fired, Red-eye reduction"
		case 0x5d:
			res.FlashMode = "Auto, Fired, Red-eye reduction, Return not detected"
		case 0x5f:
			res.FlashMode = "Auto, Fired, Red-eye reduction, Return detected"
		default:
			res.FlashMode = res.Flash
		}
	}
	if s.ExposureProgram > 0 {
		switch s.ExposureProgram {
		case 0:
			res.ExposureProgram = nil
			// res.ExposureProgram = "Not Defined"
		case 1:
			res.ExposureProgram = "Manual"
		case 2:
			res.ExposureProgram = "Program AE"
		case 3:
			res.ExposureProgram = "Aperture-priority AE"
		case 4:
			res.ExposureProgram = "Shutter speed priority AE"
		case 5:
			res.ExposureProgram = "Creative (Slow speed)"
		case 6:
			res.ExposureProgram = "Action (High speed)"
		case 7:
			res.ExposureProgram = "Portrait"
		case 8:
			res.ExposureProgram = "Landscape"
		case 9:
			res.ExposureProgram = "Bulb"
		default:
			res.ExposureProgram = s.ExposureProgram
		}
	}
	if s.MeteringMode > 0 {
		switch s.MeteringMode {
		case 0:
			res.MeteringMode = nil
			// res.MeteringMode = "Unknown"
		case 1:
			res.MeteringMode = "Average"
		case 2:
			res.MeteringMode = "Center-weighted average"
		case 3:
			res.MeteringMode = "Spot"
		case 4:
			res.MeteringMode = "Multi-spot"
		case 5:
			res.MeteringMode = "Multi-segment"
		case 6:
			res.MeteringMode = "Partial"
		case 255:
			res.MeteringMode = "Other"
		default:
			res.MeteringMode = s.MeteringMode
		}
	}
	if s.Compression > 0 {
		switch s.Compression {
		case 1:
			res.Compression = "Uncompressed"
		case 2:
			res.Compression = "CCITT 1D"
		case 3:
			res.Compression = "T4/Group 3 Fax"
		case 4:
			res.Compression = "T6/Group 4 Fax"
		case 5:
			res.Compression = "LZW"
		case 6:
			res.Compression = "JPEG (old-style)"
		case 7:
			res.Compression = "JPEG"
		case 8:
			res.Compression = "Adobe Deflate"
		case 9:
			res.Compression = "JBIG B&W"
		case 10:
			res.Compression = "JBIG Color"
		case 99:
			res.Compression = "JPEG"
		case 262:
			res.Compression = "Kodak 262"
		case 32766:
			res.Compression = "Next"
		case 32767:
			res.Compression = "Sony ARW Compressed"
		case 32769:
			res.Compression = "Packed RAW"
		case 32770:
			res.Compression = "Samsung SRW Compressed"
		case 32771:
			res.Compression = "CCIRLEW"
		case 32772:
			res.Compression = "Samsung SRW Compressed 2"
		case 32773:
			res.Compression = "PackBits"
		case 32809:
			res.Compression = "Thunderscan"
		case 32867:
			res.Compression = "Kodak KDC Compressed"
		case 32895:
			res.Compression = "IT8CTPAD"
		case 32896:
			res.Compression = "IT8LW"
		case 32897:
			res.Compression = "IT8MP"
		case 32898:
			res.Compression = "IT8BL"
		case 32908:
			res.Compression = "PixarFilm"
		case 32909:
			res.Compression = "PixarLog"
		case 32946:
			res.Compression = "Deflate"
		case 32947:
			res.Compression = "DCS"
		case 33003:
			res.Compression = "Aperio JPEG 2000 YCbCr"
		case 33005:
			res.Compression = "Aperio JPEG 2000 RGB"
		case 34661:
			res.Compression = "JBIG"
		case 34676:
			res.Compression = "SGILog"
		case 34677:
			res.Compression = "SGILog24"
		case 34712:
			res.Compression = "JPEG 2000"
		case 34713:
			res.Compression = "Nikon NEF Compressed"
		case 34715:
			res.Compression = "JBIG2 TIFF FX"
		case 34718:
			res.Compression = "Microsoft Document Imaging (MDI) Binary Level Codec"
		case 34719:
			res.Compression = "Microsoft Document Imaging (MDI) Progressive Transform Codec"
		case 34720:
			res.Compression = "Microsoft Document Imaging (MDI) Vector"
		case 34887:
			res.Compression = "ESRI Lerc"
		case 34892:
			res.Compression = "Lossy JPEG"
		case 34925:
			res.Compression = "LZMA2"
		case 34926:
			res.Compression = "Zstd"
		case 34927:
			res.Compression = "WebP"
		case 34933:
			res.Compression = "PNG"
		case 34934:
			res.Compression = "JPEG XR"
		case 65000:
			res.Compression = "Kodak DCR Compressed"
		case 65535:
			res.Compression = "Pentax PEF Compressed"
		default:
			res.Compression = s.Compression
		}
	}
	if s.ColorSpace > 0 {
		switch s.ColorSpace {
		case 0x1:
			res.ColorSpace = "sRGB"
		case 0x2:
			res.ColorSpace = "Adobe RGB"
		case 0xfffd:
			res.ColorSpace = "Wide Gamut RGB"
		case 0xfffe:
			res.ColorSpace = "ICC Profile"
		case 0xffff:
			res.ColorSpace = "Uncalibrated"
		default:
			res.ColorSpace = s.ColorSpace
		}
	}
	if s.SensingMethod > 0 {
		switch s.SensingMethod {
		case 1:
			res.SensingMethod = "Not defined"
		case 2:
			res.SensingMethod = "One-chip color area"
		case 3:
			res.SensingMethod = "Two-chip color area"
		case 4:
			res.SensingMethod = "Three-chip color area"
		case 5:
			res.SensingMethod = "Color sequential area"
		case 7:
			res.SensingMethod = "Trilinear"
		case 8:
			res.SensingMethod = "Color sequential linear"
		default:
			res.SensingMethod = s.SensingMethod
		}
	}
	if s.ExposureMode > 0 {
		switch s.ExposureMode {
		case 0:
			res.ExposureMode = "Auto"
		case 1:
			res.ExposureMode = "Manual"
		case 2:
			res.ExposureMode = "Auto bracket"
		default:
			res.ExposureMode = s.ExposureMode
		}
	}
	res.GPS = populateGPSFields(s)

	return res
}

// Populates the GPS fields
func populateGPSFields(s *bimg.EXIF) *EXIFGPS {
	if s.GPSLatitude == "" || s.GPSLongitude == "" {
		return nil
	}

	res := &EXIFGPS{}

	res.Latitude = parseGPSCoordinate(s.GPSLatitude)
	switch s.GPSLatitudeRef {
	case "N", "n":
	case "S", "s":
		res.Latitude *= -1
	default:
		return nil
	}
	res.Latitude = math.Round(res.Latitude*100_000) / 100_000

	res.Longitude = parseGPSCoordinate(s.GPSLongitude)
	switch s.GPSLongitudeRef {
	case "E", "e":
	case "W", "w":
		res.Longitude *= -1
	default:
		return nil
	}
	res.Longitude = math.Round(res.Longitude*100_000) / 100_000

	if s.GPSAltitude != "" {
		if alt, err := parseEXIFRational(s.GPSAltitude); err == nil {
			if s.GPSAltitudeRef == "1" {
				alt *= -1
			}
			res.Altitude = formatFloat(alt, 0) + " m"
		}
	}

	if s.GPSSpeed != "" && s.GPSSpeedRef != "" {
		if speed, err := parseEXIFRational(s.GPSSpeed); err == nil {
			res.Speed = formatFloat(speed, 2)
			switch s.GPSSpeedRef {
			case "K", "k":
				res.Speed += " km/h"
			case "M", "m":
				res.Speed += " mph"
			case "N", "n":
				res.Speed += " kn"
			}
		}
	}

	if s.GPSImgDirection != "" {
		if dir, err := parseEXIFRational(s.GPSImgDirection); err == nil {
			res.Direction = math.Round(dir*100) / 100

			switch s.GPSImgDirectionRef {
			case "T", "t":
				res.DirectionRef = "True North"
			case "M", "m":
				res.DirectionRef = "Magnetic North"
			}
		}
	}

	return res
}

// Parses a GPS coordinate (latitude or longitude) converting from the degrees, minutes, and seconds, into a decimal
func parseGPSCoordinate(v string) float64 {
	parts := strings.Split(v, " ")
	var r, res float64
	var err error
	for i, p := range parts {
		r, err = parseEXIFRational(p)
		if err != nil {
			log.Printf("Failed to parse EXIF rational value in GPS coordinates '%v': '%v'", v, err)
			return 0
		}
		res += r / math.Pow(60, float64(i))
	}
	return res
}

// Formats a rational so it's in a clearer format for humans
// Inspired by https://stackoverflow.com/a/24205621/192024
func formatHumanRational(val string) string {
	f := parseEXIFRationalOrLogError(val)
	if f == 0 {
		return "0"
	}
	if f >= 0.3 {
		return formatFloat(f, 2)
	}

	den := int(0.5 + 1/f)
	return fmt.Sprintf("1/%d", den)
}

// Formats the values for XResolution and YResolution
func formatResolution(resolution string, resolutionUnit int) string {
	v := formatFloat(parseEXIFRationalOrLogError(resolution), 2)
	switch resolutionUnit {
	case 2: // inches
		v += " ppi"
	case 3: // cms
		v += " ppcm"
	}
	return v
}

// Formats the DateTime values
func formatDateTime(orig string) string {
	t, err := time.Parse("2006:01:02 15:04:05", orig)
	if err != nil {
		return ""
	}

	// Format like RFC3339, but without any time zone
	return t.Format("2006-01-02T15:04:05")
}

// Formats a float with at most n decimal digits
func formatFloat(v float64, n int) string {
	return strings.TrimRight(
		strings.TrimRight(
			strconv.FormatFloat(v, 'f', 2, 64),
			"0"),
		".")
}

// Parse a rational number (represented as a fractional in a string) and returns a float value
func parseEXIFRational(v string) (float64, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, nil
	}

	var (
		num float64
		den float64
		err error
	)

	parts := strings.SplitN(v, "/", 3)
	if len(parts) > 2 {
		return 0, errors.New("invalid value: more than 1 slash found")
	}

	if parts[0] == "" {
		return 0, errors.New("invalid value: numerator is empty")
	}
	num, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	if len(parts) == 1 || num == 0 {
		return num, nil
	} else {
		den, err = strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, err
		}
		if den == 0 {
			return 0, errors.New("invalid value: denumerator is 0")
		}
		return num / den, nil
	}
}

// Like parseEXIFRational, but logs errors and returns a single value
func parseEXIFRationalOrLogError(v string) float64 {
	res, err := parseEXIFRational(v)
	if err != nil {
		log.Printf("Failed to parse EXIF rational value '%v': '%v'", v, err)
	}
	return res
}
