package main

import (
    "flag"
    "fmt"
    "os"
    "os/exec"
    "strconv"
)

func main() {
    // Define flags for -probesize and -analyzeduration
    var probeSize string
    var analyzeDuration string

    flag.StringVar(&probeSize, "probesize", "", "Set the probe size (default: reduce by factor of 10)")
    flag.StringVar(&analyzeDuration, "analyzeduration", "", "Set the analyze duration (default: reduce by factor of 10)")

    // Parse the command line arguments
    flag.Parse()

    // Get the ffprobe command and its arguments
    args := flag.Args()
    if len(args) == 0 {
        fmt.Println("No ffprobe command provided.")
        os.Exit(1)
    }

    // Adjust -probesize and -analyzeduration if provided
    if probeSize != "" {
        if adjusted, err := reduceByFactor(probeSize, 10); err == nil {
            probeSize = adjusted
        } else {
            fmt.Fprintf(os.Stderr, "Invalid probesize value: %v\n", err)
            os.Exit(1)
        }
    } else {
        probeSize = "500000" // Default reduced value
    }

    if analyzeDuration != "" {
        if adjusted, err := reduceByFactor(analyzeDuration, 10); err == nil {
            analyzeDuration = adjusted
        } else {
            fmt.Fprintf(os.Stderr, "Invalid analyzeduration value: %v\n", err)
            os.Exit(1)
        }
    } else {
        analyzeDuration = "500000" // Default reduced value
    }

    // Construct the command to execute the real ffprobe
    ffprobeArgs := []string{"-probesize", probeSize, "-analyzeduration", analyzeDuration}
    ffprobeArgs = append(ffprobeArgs, args...)

    // Execute the real ffprobe
    cmd := exec.Command("ffprobe-real", ffprobeArgs...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error executing ffprobe: %v\n", err)
        os.Exit(1)
    }
}

// reduceByFactor reduces a numeric string by the given factor
func reduceByFactor(value string, factor int) (string, error) {
    num, err := strconv.Atoi(value)
    if err != nil {
        return "", err
    }
    return strconv.Itoa(num / factor), nil
}