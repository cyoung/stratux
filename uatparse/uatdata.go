package uatparse

// AIRMET = AIRMET/SIGMET/ (TFR?)

const (
	UATMSG_TEXT   = 1
	UATMSG_NEXRAD = 2
	UATMSG_AIRMET = 3 // AIRMET. Decoded.

	// How the coordinates should be used in a graphical AIRMET.
	AIRMET_POLYGON = 1
	AIRMET_ELLIPSE = 2
	AIRMET_PRISM   = 3
	AIRMET_3D      = 4
)

// Points can be in 3D - take care that altitude is used correctly.
type GeoPoint struct {
	Lat float64
	Lon float64
	Alt int32
}

type UATAirmet struct {
	Points []GeoPoint // Points
}

type UATMsgDecoded struct {
	Type int
}
