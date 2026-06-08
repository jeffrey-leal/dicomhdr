package main

// dicomMagicOffset is the byte offset of the DICOM magic bytes.
const dicomMagicOffset = 128

// dicomMagic is the four-byte signature present in every valid DICOM file.
const dicomMagic = "DICM"

// hasDICMSignature reports whether buf carries the "DICM" signature at byte
// offset 128. buf must be at least dicomMagicOffset+len(dicomMagic) bytes long.
func hasDICMSignature(buf []byte) bool {
	if len(buf) < dicomMagicOffset+len(dicomMagic) {
		return false
	}
	for i := 0; i < len(dicomMagic); i++ {
		if buf[dicomMagicOffset+i] != dicomMagic[i] {
			return false
		}
	}
	return true
}
