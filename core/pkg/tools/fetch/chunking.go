package fetch

import "fmt"

func applyChunking(content string, startIndex, maxLength int) string {
	originalLength := len(content)

	if startIndex >= originalLength {
		return "<error>No more content available.</error>"
	}

	endIndex := startIndex + maxLength
	if endIndex > originalLength {
		endIndex = originalLength
	}

	truncatedContent := content[startIndex:endIndex]
	if truncatedContent == "" {
		return "<error>No more content available.</error>"
	}

	actualContentLength := len(truncatedContent)
	remainingContent := originalLength - (startIndex + actualContentLength)

	if actualContentLength == maxLength && remainingContent > 0 {
		nextStart := startIndex + actualContentLength
		truncatedContent += fmt.Sprintf("\n\n<error>Content truncated. Call the fetch tool with a start_index of %d to get more content.</error>", nextStart)
	}

	return truncatedContent
}
