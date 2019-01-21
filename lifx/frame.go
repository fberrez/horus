package lifx

// These variables represent frame settings.
// The first characters determine their status, see:
// * N = Not
// * T = Tagged
// * A = Addressable
// See documentation about frame settings here:
// https://lan.developer.lifx.com/docs/header-description#frame
var (
	// NTNAFrame represents settings with `tagged` and `addressable` set to `false`.
	NTNAFrame = [2]byte{0X00, 0X04}
	// NTAFrame represents settings with `tagged` set to `false` and `addressable` set to `true`.
	NTAFrame = [2]byte{0X00, 0X14}
	// TNAFrame represents settings with `tagged` set to `true` and `addressable` set to `false`.
	TNAFrame = [2]byte{0X00, 0X24}
	// TAFrame represents settings with `tagged` and `addressable` set to `true`.
	TAFrame = [2]byte{0X00, 0X34}
)
