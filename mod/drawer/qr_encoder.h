#ifndef NICKELDRAWER_QR_ENCODER_H
#define NICKELDRAWER_QR_ENCODER_H

// Small QR Code encoder for the `qr` and `pair` components: byte mode, error
// correction level M, versions 1 to 20 (up to 666 bytes; ui.json caps a qr
// value at 200 characters). Written for this project from ISO/IEC 18004; the
// function-pattern and Reed-Solomon steps follow the well-known structure of
// Nayuki's MIT-licensed "QR Code generator" (nayuki.io), no code copied. The
// mask is chosen with Table 11 read literally, as the segno library does.
// No Qt and no C++ standard library: tests/qr_encoder_test.cc checks whole
// matrices against segno (tests/qr_vectors.inc).

namespace QrEncoder {

static const int MIN_VERSION = 1;
static const int MAX_VERSION = 20;
static const int MAX_SIDE = 4 * MAX_VERSION + 17;
static const int AUTOMATIC_MASK = -1;

struct QrCode {
    int version;
    int side; // modules per row, without the quiet zone
    int mask;
    unsigned char modules[MAX_SIDE * MAX_SIDE]; // 1 = dark, row-major

    bool isDark(int x, int y) const { return modules[y * side + x] != 0; }
};

// Encodes bytes into the smallest version that fits. forcedMask 0..7 skips
// the penalty-based mask choice (tests compare single masks with it).
// Returns false when the data is too long or the arguments are invalid.
bool encode(const unsigned char *data, int length, QrCode *code, int forcedMask = AUTOMATIC_MASK);

} // namespace QrEncoder

#endif
