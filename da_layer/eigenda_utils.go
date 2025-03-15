package da

import (
	"log"
	"time"
)

// Helper function for retries
func retry(attempts int, sleep time.Duration, f func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = f()
		if err == nil {
			return nil
		}

		if i < attempts-1 {
			log.Printf("Attempt %d failed: %v. Retrying in %v...", i+1, err, sleep)
			time.Sleep(sleep)
			// Exponential backoff
			sleep = sleep * 2
		}
	}
	return err
}

// removeNullBytesPadding removes null bytes padding from the end of the data
func (s *DataAvailabilityService) removeNullBytesPadding(data []byte) []byte {
	// Find the first non-null byte from the beginning
	var startPos int
	for startPos = 0; startPos < len(data); startPos++ {
		if data[startPos] != 0 {
			break
		}
	}

	// Find the last non-null byte from the end
	var endPos int
	for endPos = len(data) - 1; endPos >= 0; endPos-- {
		if data[endPos] != 0 {
			break
		}
	}

	// If the data is all null bytes, return empty
	if startPos > endPos {
		return []byte{}
	}

	// Return the data between the first and last non-null bytes
	return data[startPos : endPos+1]
}
