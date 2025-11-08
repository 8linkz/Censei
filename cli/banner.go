package cli

import "fmt"

// PrintBanner displays the ASCII art logo when the application starts
func PrintBanner() {
	banner := `
 _____                          _
/  __ \                        (_)
| /  \/  ___  _ __   ___   ___  _
| |     / _ \| '_ \ / __| / _ \| |
| \__/\|  __/| | | |\__ \|  __/| |
 \____/ \___||_| |_||___/ \___||_|

`
	fmt.Println(banner)
	fmt.Println("                     Censys Index Scanner - v0.8")
}

// PrintBannerWithMode displays the ASCII art logo with API mode indication
func PrintBannerWithMode(isLegacyMode bool) {
	banner := `
 _____                          _
/  __ \                        (_)
| /  \/  ___  _ __   ___   ___  _
| |     / _ \| '_ \ / __| / _ \| |
| \__/\|  __/| | | |\__ \|  __/| |
 \____/ \___||_| |_||___/ \___||_|

`
	fmt.Println(banner)
	fmt.Println("                     Censys Index Scanner - v0.8")

	if isLegacyMode {
		fmt.Println("                        Mode: Legacy CLI")
	} else {
		fmt.Println("                       Mode: Platform API v3")
	}
}
