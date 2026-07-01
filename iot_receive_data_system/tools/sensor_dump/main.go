package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type dbConfig struct {
	host     string
	port     string
	user     string
	password string
	dbname   string
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "dump":
		err = runDump(os.Args[2:])
	case "restore":
		err = runRestore(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "sensor-dump: %v\n", err)
		os.Exit(1)
	}
}

func runDump(args []string) error {
	fs := flag.NewFlagSet("dump", flag.ContinueOnError)
	cfg := addDBFlags(fs)
	out := fs.String("out", defaultDumpPath(), "output .tar.gz path")
	jobs := fs.Int("jobs", max(2, runtime.NumCPU()), "parallel pg_dump jobs")
	compressLevel := fs.Int("compress", 6, "pg_dump per-file compression level 0-9")
	withSensors := fs.Bool("with-sensors", true, "include sensors table for sensor_id mapping")
	withoutSensors := fs.Bool("without-sensors", false, "dump only sensor_datas and sensor_thermal_frames")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *withoutSensors {
		*withSensors = false
	}
	if *jobs < 1 {
		return errors.New("--jobs must be >= 1")
	}
	if *compressLevel < 0 || *compressLevel > 9 {
		return errors.New("--compress must be between 0 and 9")
	}
	if strings.TrimSpace(*out) == "" {
		return errors.New("--out is required")
	}

	outputPath, err := filepath.Abs(*out)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	workDir, err := os.MkdirTemp(filepath.Dir(outputPath), ".sensor-dump-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	dumpDir := filepath.Join(workDir, "pgdump")
	patterns := dumpTablePatterns(*withSensors)
	pgArgs := []string{
		"-h", cfg.host,
		"-p", cfg.port,
		"-U", cfg.user,
		"-d", cfg.dbname,
		"-Fd",
		"--data-only",
		"--no-owner",
		"--no-acl",
		"-j", fmt.Sprintf("%d", *jobs),
		"-Z", fmt.Sprintf("%d", *compressLevel),
		"-f", dumpDir,
	}
	for _, pattern := range patterns {
		pgArgs = append(pgArgs, "-t", pattern)
	}

	fmt.Printf("dumping tables: %s\n", strings.Join(patterns, ", "))
	if err := runCommand(cfg, "pg_dump", pgArgs...); err != nil {
		return err
	}

	manifest := []byte(fmt.Sprintf("created_at=%s\nformat=pg_dump_directory_tar_gz\ndata_only=true\njobs=%d\ntables=%s\n",
		time.Now().UTC().Format(time.RFC3339),
		*jobs,
		strings.Join(patterns, ","),
	))
	if err := os.WriteFile(filepath.Join(workDir, "MANIFEST.txt"), manifest, 0o644); err != nil {
		return err
	}

	if err := tarGzipDir(workDir, outputPath); err != nil {
		return err
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		return err
	}
	fmt.Printf("dump complete: %s (%s)\n", outputPath, humanBytes(info.Size()))
	return nil
}

func runRestore(args []string) error {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	cfg := addDBFlags(fs)
	in := fs.String("in", "", "input .tar.gz path")
	jobs := fs.Int("jobs", max(2, runtime.NumCPU()), "parallel pg_restore jobs")
	truncate := fs.Bool("truncate", true, "truncate sensors/sensor_datas/sensor_thermal_frames before restore")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *jobs < 1 {
		return errors.New("--jobs must be >= 1")
	}
	if strings.TrimSpace(*in) == "" {
		return errors.New("--in is required")
	}
	inputPath, err := filepath.Abs(*in)
	if err != nil {
		return err
	}
	if _, err := os.Stat(inputPath); err != nil {
		return err
	}

	workDir, err := os.MkdirTemp("", "sensor-restore-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	if err := untarGzip(inputPath, workDir); err != nil {
		return err
	}
	dumpDir := filepath.Join(workDir, "pgdump")
	if _, err := os.Stat(dumpDir); err != nil {
		return fmt.Errorf("pgdump directory not found in archive: %w", err)
	}

	if *truncate {
		sql := `TRUNCATE TABLE sensor_thermal_frames, sensor_datas, sensors RESTART IDENTITY CASCADE;`
		if err := runCommand(cfg, "psql", "-h", cfg.host, "-p", cfg.port, "-U", cfg.user, "-d", cfg.dbname, "-v", "ON_ERROR_STOP=1", "-c", sql); err != nil {
			return err
		}
	}

	pgArgs := []string{
		"-h", cfg.host,
		"-p", cfg.port,
		"-U", cfg.user,
		"-d", cfg.dbname,
		"--data-only",
		"--no-owner",
		"--no-acl",
		"-j", fmt.Sprintf("%d", *jobs),
		dumpDir,
	}
	if err := runCommand(cfg, "pg_restore", pgArgs...); err != nil {
		return err
	}
	fmt.Println("restore complete")
	return nil
}

func addDBFlags(fs *flag.FlagSet) dbConfig {
	cfg := dbConfig{}
	fs.StringVar(&cfg.host, "host", envDefault("PG_HOST", "irds_db_postgresql_v1"), "PostgreSQL host")
	fs.StringVar(&cfg.port, "port", envDefault("PG_PORT", "5432"), "PostgreSQL port")
	fs.StringVar(&cfg.user, "user", envDefault("PG_USER", "postgres"), "PostgreSQL user")
	fs.StringVar(&cfg.password, "password", envDefault("PG_PASSWORD", ""), "PostgreSQL password")
	fs.StringVar(&cfg.dbname, "db", envDefault("PG_DB_NAME", "sg_tch_v3"), "PostgreSQL database")
	return cfg
}

func dumpTablePatterns(withSensors bool) []string {
	patterns := []string{}
	if withSensors {
		patterns = append(patterns, "public.sensors")
	}
	return append(patterns, "public.sensor_datas*", "public.sensor_thermal_frames*")
}

func runCommand(cfg dbConfig, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if cfg.password != "" {
		cmd.Env = append(cmd.Env, "PGPASSWORD="+cfg.password)
	}
	fmt.Printf("running: %s %s\n", name, strings.Join(args, " "))
	return cmd.Run()
}

func tarGzipDir(srcDir, outPath string) error {
	tmpPath := outPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	gzw, err := gzip.NewWriterLevel(out, gzip.BestSpeed)
	if err != nil {
		_ = out.Close()
		return err
	}
	tw := tar.NewWriter(gzw)

	walkErr := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == srcDir {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(tw, in)
		return err
	})
	closeErr := errors.Join(tw.Close(), gzw.Close(), out.Close())
	if err := errors.Join(walkErr, closeErr); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, outPath)
}

func untarGzip(inPath, outDir string) error {
	in, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer in.Close()

	gzr, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(outDir, filepath.Clean(header.Name))
		if !strings.HasPrefix(target, filepath.Clean(outDir)+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe archive path: %s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if err := errors.Join(copyErr, closeErr); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported archive entry type %d: %s", header.Typeflag, header.Name)
		}
	}
}

func defaultDumpPath() string {
	return fmt.Sprintf("/dumps/sensor_analysis_%s.tar.gz", time.Now().Format("20060102_150405"))
}

func envDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func humanBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage:
  sensor-dump dump    --out /dumps/sensor_analysis.tar.gz --jobs 8
  sensor-dump restore --in  /dumps/sensor_analysis.tar.gz --jobs 8 --truncate

Environment defaults:
  PG_HOST, PG_PORT, PG_USER, PG_PASSWORD, PG_DB_NAME`)
}
