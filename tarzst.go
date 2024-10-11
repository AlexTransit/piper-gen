package main

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
)

type TarZstWriter struct {
	file  *os.File
	zstWr *zstd.Encoder
	tarWr *tar.Writer
}

func (tzw *TarZstWriter) Append(h *tar.Header, r io.Reader) error {
	if err := tzw.tarWr.WriteHeader(h); err != nil {
		return fmt.Errorf("TarZstWriter.Append: header: %w", err)
	}
	if _, err := io.Copy(tzw.tarWr, r); err != nil {
		return fmt.Errorf("TarZstWriter.Append: copy: %w", err)
	}
	if err := tzw.tarWr.Flush(); err != nil {
		return fmt.Errorf("TarZstWriter.Append: copy: %w", err)
	}
	return nil
}

func (tzw *TarZstWriter) AppendFile(dstPth, srcFn string) error {
	f, err := os.Open(srcFn)
	if err != nil {
		return fmt.Errorf("TarZstWriter.AppendFile: open `%s`: %w", srcFn, err)
	}
	defer f.Close()

	fi, err := os.Lstat(srcFn)
	if err != nil {
		return fmt.Errorf("TarZstWriter.AppendFile: stat: %w", err)
	}
	h := &tar.Header{
		Name: dstPth,
		Mode: int64(fi.Mode()),
		Size: fi.Size(),
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		nm, err := os.Readlink(srcFn)
		if err != nil {
			return fmt.Errorf("TarZstWriter.AppendFile: read symlink: %w", err)
		}
		h.Linkname = nm
	}
	if err := tzw.Append(h, f); err != nil {
		return fmt.Errorf("TarZstWriter.AppendFile: %w", err)
	}
	return nil
}

func (tzw *TarZstWriter) Close() (err error) {
	te := tzw.tarWr.Close()
	ze := tzw.zstWr.Close()
	fe := tzw.file.Close()
	if te != nil {
		err = errors.Join(fmt.Errorf("TarZstWriter.Close: tar: %w", te))
	}
	if ze != nil {
		err = errors.Join(fmt.Errorf("TarZstWriter.Close: zst: %w", ze))
	}
	if fe != nil {
		err = errors.Join(fmt.Errorf("TarZstWriter.Close: file: %w", fe))
	}
	return err
}

func createTarZst(fn string, opts ...zstd.EOption) (*TarZstWriter, error) {
	if opts == nil {
		opts = []zstd.EOption{
			zstd.WithEncoderLevel(zstd.SpeedBestCompression),
		}
	}

	os.MkdirAll(filepath.Dir(fn), 0755)
	file, err := os.Create(fn)
	if err != nil {
		return nil, fmt.Errorf("createTarZst: create output file: %w", err)
	}

	zstWr, err := zstd.NewWriter(file, opts...)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("createTarZst: create zstd writer: %w", err)
	}

	tzw := &TarZstWriter{
		file:  file,
		zstWr: zstWr,
		tarWr: tar.NewWriter(zstWr),
	}
	return tzw, nil
}
