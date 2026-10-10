// Host unit tests for qr_encoder against matrices made by the segno library
// (tests/qr_vectors.inc, byte mode, level M, no ECC boost). Run:
//   make -C mod/drawer docker-test

#include "../qr_encoder.h"
#include "../../common/test_expect.h"

struct QrVector {
    const char *text;
    int forcedMask; // -1: the encoder chooses by penalty
    int version;
    int mask;
    const char *const *rows;
};

#include "qr_vectors.inc"

static QrEncoder::QrCode code; // ~9 KB, kept off the stack

static int mismatchedModules(const QrVector &vector)
{
    int mismatches = 0;
    for (int y = 0; y < code.side; y++) {
        for (int x = 0; x < code.side; x++) {
            bool expectedDark = vector.rows[y][x] == '1';
            if (expectedDark != code.isDark(x, y)) {
                mismatches++;
            }
        }
    }
    return mismatches;
}

static void testVectors()
{
    for (unsigned i = 0; i < sizeof(VECTORS) / sizeof(VECTORS[0]); i++) {
        const QrVector &vector = VECTORS[i];
        bool isEncoded = QrEncoder::encode((const unsigned char *)vector.text, (int)strlen(vector.text), &code,
                                           vector.forcedMask);
        EXPECT_EQ(true, isEncoded);
        if (!isEncoded) {
            continue;
        }
        EXPECT_EQ(vector.version, code.version);
        EXPECT_EQ(4 * vector.version + 17, code.side);
        EXPECT_EQ((int)strlen(vector.rows[0]), code.side);
        EXPECT_EQ(vector.mask, code.mask);
        int mismatches = mismatchedModules(vector);
        if (mismatches) {
            printf("vector %u (%s): %d modules differ\n", i, vector.text, mismatches);
        }
        EXPECT_EQ(0, mismatches);
    }
}

static void testLimits()
{
    static unsigned char data[700];
    memset(data, 'a', sizeof(data));
    EXPECT_EQ(true, QrEncoder::encode(data, 666, &code)); // version 20-M holds 666 bytes
    EXPECT_EQ(20, code.version);
    EXPECT_EQ(false, QrEncoder::encode(data, 667, &code));
    EXPECT_EQ(true, QrEncoder::encode(data, 0, &code));
    EXPECT_EQ(1, code.version);
    EXPECT_EQ(false, QrEncoder::encode(data, 4, &code, 8));
    EXPECT_EQ(false, QrEncoder::encode(0, 4, &code));
    EXPECT_EQ(false, QrEncoder::encode(data, 4, 0));
}

int main()
{
    testVectors();
    testLimits();
    return finishTests("qr_encoder");
}
