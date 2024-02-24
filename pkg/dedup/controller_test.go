package dedup

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestReceiveLSOF(t *testing.T) {
	pattern := regexp.MustCompile(`^\S+\s+\S+\s+\S+\s+\S+\s+REG\s+\S+\s+(\S+)\s+(\S+)$`)
	data := "python3 855 root  mem    REG  0,109        131  57344563 /usr/lib/locale/C.UTF-8/LC_ADDRESS\npython3 855 root  mem    REG  0,109         23  57344567 /usr/lib/locale/C.UTF-8/LC_MEASUREMENT\npython3 855 root    0r  FIFO   0,13        0t0   3059366 pipe"
	lines := strings.Split(data, "\n")
	result := make([]string, 2)
	for _, line := range lines {
		if pattern.MatchString(line) {
			match := pattern.FindStringSubmatch(line)
			t.Logf("%v", match)
			result = append(result, fmt.Sprint(match[1], match[2]))
		}
	}
	t.Logf("%v", lines[0])
	if len(result) == 0 {
		t.Error("Empty result")
	}
	if result[0] != "57344563 /usr/lib/locale/C.UTF-8/LC_ADDRESS" || result[1] != "57344567 /usr/lib/locale/C.UTF-8/LC_MEASUREMENT" {
		t.Errorf("wrong: %v", result)
	}
}
