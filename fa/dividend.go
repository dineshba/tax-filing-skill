package fa

import (
	"encoding/json"
	"fmt"
	"time"
)

// DividendEvent is a dividend paid by the foreign entity, used to attribute
// dividend income to each lot based on shares held on the ex-date.
type DividendEvent struct {
	ExDate      time.Time
	USDPerShare float64
}

// UnmarshalJSON parses {"exDate":"2025-02-19","usdPerShare":0.83}.
func (d *DividendEvent) UnmarshalJSON(b []byte) error {
	var raw struct {
		ExDate      string  `json:"exDate"`
		USDPerShare float64 `json:"usdPerShare"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	t, err := time.Parse("2006-01-02", raw.ExDate)
	if err != nil {
		return fmt.Errorf("dividend exDate %q: %w", raw.ExDate, err)
	}
	d.ExDate = t
	d.USDPerShare = raw.USDPerShare
	return nil
}
