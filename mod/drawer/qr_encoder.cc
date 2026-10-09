#include "qr_encoder.h"

#include <string.h>

namespace QrEncoder {

// Error correction level M, indexed by version (index 0 unused).
static const int ECC_CODEWORDS_PER_BLOCK[MAX_VERSION + 1] = {
    0, 10, 16, 26, 18, 24, 16, 18, 22, 22, 26, 30, 22, 22, 24, 24, 28, 28, 26, 26, 26,
};
static const int ERROR_CORRECTION_BLOCKS[MAX_VERSION + 1] = {
    0, 1, 1, 1, 2, 2, 4, 4, 4, 5, 5, 5, 8, 9, 9, 10, 10, 11, 13, 14, 16,
};
static const int LEVEL_M_FORMAT_BITS = 0;
static const int BYTE_MODE_INDICATOR = 0x4;
static const int MAX_CODEWORDS = 1200; // version 20 has 1085
static const int MAX_ECC_LENGTH = 30;
static const int MASK_COUNT = 8;

// Penalty weights of ISO/IEC 18004 section 8.8.2.
static const int PENALTY_RUN = 3;
static const int PENALTY_BOX = 3;
static const int PENALTY_FINDER = 40;
static const int PENALTY_BALANCE = 10;

struct Work {
    QrCode *code;
    unsigned char isFunction[MAX_SIDE * MAX_SIDE];
};

static bool bitOf(int value, int index)
{
    return ((value >> index) & 1) != 0;
}

static void setModule(Work *work, int x, int y, bool isDark)
{
    work->code->modules[y * work->code->side + x] = isDark ? 1 : 0;
}

static void setFunctionModule(Work *work, int x, int y, bool isDark)
{
    setModule(work, x, y, isDark);
    work->isFunction[y * work->code->side + x] = 1;
}

static bool isFunctionModule(const Work *work, int x, int y)
{
    return work->isFunction[y * work->code->side + x] != 0;
}

// ---- Capacity ----

static int rawDataModules(int version)
{
    int result = (16 * version + 128) * version + 64;
    if (version >= 2) {
        int alignmentCount = version / 7 + 2;
        result -= (25 * alignmentCount - 10) * alignmentCount - 55;
        if (version >= 7) {
            result -= 36;
        }
    }
    return result;
}

static int dataCodewords(int version)
{
    return rawDataModules(version) / 8 - ECC_CODEWORDS_PER_BLOCK[version] * ERROR_CORRECTION_BLOCKS[version];
}

static int countBits(int version)
{
    return version <= 9 ? 8 : 16;
}

static int smallestVersionFor(int length)
{
    for (int version = MIN_VERSION; version <= MAX_VERSION; version++) {
        if (4 + countBits(version) + 8 * length <= dataCodewords(version) * 8) {
            return version;
        }
    }
    return 0;
}

// ---- Data codewords ----

struct BitBuffer {
    unsigned char *bytes;
    int bitLength;
};

static void appendBits(BitBuffer *buffer, int value, int count)
{
    for (int i = count - 1; i >= 0; i--) {
        if (bitOf(value, i)) {
            buffer->bytes[buffer->bitLength >> 3] |= (unsigned char)(0x80 >> (buffer->bitLength & 7));
        }
        buffer->bitLength++;
    }
}

static void buildDataCodewords(const unsigned char *data, int length, int version, unsigned char *codewords)
{
    int capacityBits = dataCodewords(version) * 8;
    memset(codewords, 0, dataCodewords(version));
    BitBuffer buffer = {codewords, 0};
    appendBits(&buffer, BYTE_MODE_INDICATOR, 4);
    appendBits(&buffer, length, countBits(version));
    for (int i = 0; i < length; i++) {
        appendBits(&buffer, data[i], 8);
    }
    int terminator = capacityBits - buffer.bitLength < 4 ? capacityBits - buffer.bitLength : 4;
    appendBits(&buffer, 0, terminator);
    appendBits(&buffer, 0, (8 - buffer.bitLength % 8) % 8);
    for (int pad = 0xEC; buffer.bitLength < capacityBits; pad ^= 0xEC ^ 0x11) {
        appendBits(&buffer, pad, 8);
    }
}

// ---- Reed-Solomon over GF(2^8) with the polynomial 0x11D ----

static int multiply(int x, int y)
{
    int product = 0;
    for (int i = 7; i >= 0; i--) {
        product = (product << 1) ^ ((product >> 7) * 0x11D);
        product ^= ((y >> i) & 1) * x;
    }
    return product & 0xFF;
}

static void computeDivisor(int degree, unsigned char *divisor)
{
    memset(divisor, 0, degree);
    divisor[degree - 1] = 1;
    int root = 1;
    for (int i = 0; i < degree; i++) {
        for (int j = 0; j < degree; j++) {
            divisor[j] = (unsigned char)multiply(divisor[j], root);
            if (j + 1 < degree) {
                divisor[j] ^= divisor[j + 1];
            }
        }
        root = multiply(root, 0x02);
    }
}

static void computeRemainder(const unsigned char *data, int length, const unsigned char *divisor, int degree,
                             unsigned char *remainder)
{
    memset(remainder, 0, degree);
    for (int i = 0; i < length; i++) {
        int factor = data[i] ^ remainder[0];
        memmove(remainder, remainder + 1, degree - 1);
        remainder[degree - 1] = 0;
        for (int j = 0; j < degree; j++) {
            remainder[j] ^= (unsigned char)multiply(divisor[j], factor);
        }
    }
}

// Splits data into blocks, appends each block's ECC and interleaves them.
static void addEccAndInterleave(const unsigned char *data, int version, unsigned char *result)
{
    int blockCount = ERROR_CORRECTION_BLOCKS[version];
    int eccLength = ECC_CODEWORDS_PER_BLOCK[version];
    int rawCodewords = rawDataModules(version) / 8;
    int shortBlockCount = blockCount - rawCodewords % blockCount;
    int shortBlockLength = rawCodewords / blockCount;
    int stride = shortBlockLength + 1;

    static unsigned char blocks[MAX_CODEWORDS + 64];
    unsigned char divisor[MAX_ECC_LENGTH];
    computeDivisor(eccLength, divisor);
    for (int block = 0, offset = 0; block < blockCount; block++) {
        int dataLength = shortBlockLength - eccLength + (block < shortBlockCount ? 0 : 1);
        unsigned char *target = blocks + block * stride;
        memset(target, 0, stride);
        memcpy(target, data + offset, dataLength);
        offset += dataLength;
        // Short blocks keep a padding byte at dataLength so the ECC columns line up.
        computeRemainder(target, dataLength, divisor, eccLength, target + stride - eccLength);
    }
    int written = 0;
    for (int column = 0; column < stride; column++) {
        for (int block = 0; block < blockCount; block++) {
            bool isPadding = column == shortBlockLength - eccLength && block < shortBlockCount;
            if (!isPadding) {
                result[written++] = blocks[block * stride + column];
            }
        }
    }
}

// ---- Function patterns ----

static int alignmentPositions(int version, int side, int *positions)
{
    if (version == 1) {
        return 0;
    }
    int count = version / 7 + 2;
    int step = (version * 4 + count * 2 + 1) / (count * 2 - 2) * 2;
    positions[0] = 6;
    for (int i = count - 1, position = side - 7; i >= 1; i--, position -= step) {
        positions[i] = position;
    }
    return count;
}

static int chebyshevDistance(int dx, int dy)
{
    int ax = dx < 0 ? -dx : dx;
    int ay = dy < 0 ? -dy : dy;
    return ax > ay ? ax : ay;
}

static void drawFinder(Work *work, int centerX, int centerY)
{
    int side = work->code->side;
    for (int dy = -4; dy <= 4; dy++) {
        for (int dx = -4; dx <= 4; dx++) {
            int x = centerX + dx;
            int y = centerY + dy;
            if (x >= 0 && x < side && y >= 0 && y < side) {
                int distance = chebyshevDistance(dx, dy);
                setFunctionModule(work, x, y, distance != 2 && distance != 4);
            }
        }
    }
}

static void drawAlignment(Work *work, int centerX, int centerY)
{
    for (int dy = -2; dy <= 2; dy++) {
        for (int dx = -2; dx <= 2; dx++) {
            setFunctionModule(work, centerX + dx, centerY + dy, chebyshevDistance(dx, dy) != 1);
        }
    }
}

static void drawFormatBits(Work *work, int mask)
{
    int side = work->code->side;
    int data = LEVEL_M_FORMAT_BITS << 3 | mask;
    int remainder = data;
    for (int i = 0; i < 10; i++) {
        remainder = (remainder << 1) ^ ((remainder >> 9) * 0x537);
    }
    int bits = (data << 10 | remainder) ^ 0x5412;
    for (int i = 0; i <= 5; i++) {
        setFunctionModule(work, 8, i, bitOf(bits, i));
    }
    setFunctionModule(work, 8, 7, bitOf(bits, 6));
    setFunctionModule(work, 8, 8, bitOf(bits, 7));
    setFunctionModule(work, 7, 8, bitOf(bits, 8));
    for (int i = 9; i < 15; i++) {
        setFunctionModule(work, 14 - i, 8, bitOf(bits, i));
    }
    for (int i = 0; i < 8; i++) {
        setFunctionModule(work, side - 1 - i, 8, bitOf(bits, i));
    }
    for (int i = 8; i < 15; i++) {
        setFunctionModule(work, 8, side - 15 + i, bitOf(bits, i));
    }
    setFunctionModule(work, 8, side - 8, true); // the dark module
}

static void drawVersionBits(Work *work)
{
    int version = work->code->version;
    if (version < 7) {
        return;
    }
    int remainder = version;
    for (int i = 0; i < 12; i++) {
        remainder = (remainder << 1) ^ ((remainder >> 11) * 0x1F25);
    }
    int bits = version << 12 | remainder;
    for (int i = 0; i < 18; i++) {
        int a = work->code->side - 11 + i % 3;
        int b = i / 3;
        setFunctionModule(work, a, b, bitOf(bits, i));
        setFunctionModule(work, b, a, bitOf(bits, i));
    }
}

static void drawFunctionPatterns(Work *work)
{
    int side = work->code->side;
    for (int i = 0; i < side; i++) {
        setFunctionModule(work, 6, i, i % 2 == 0);
        setFunctionModule(work, i, 6, i % 2 == 0);
    }
    drawFinder(work, 3, 3);
    drawFinder(work, side - 4, 3);
    drawFinder(work, 3, side - 4);
    int positions[8];
    int count = alignmentPositions(work->code->version, side, positions);
    for (int i = 0; i < count; i++) {
        for (int j = 0; j < count; j++) {
            bool isFinderCorner = (i == 0 && j == 0) || (i == 0 && j == count - 1) || (i == count - 1 && j == 0);
            if (!isFinderCorner) {
                drawAlignment(work, positions[i], positions[j]);
            }
        }
    }
    drawFormatBits(work, 0); // placeholder, so the modules count as function modules
    drawVersionBits(work);
}

// Places the codeword bits in the zig-zag order, skipping function modules.
static void drawCodewords(Work *work, const unsigned char *codewords, int count)
{
    int side = work->code->side;
    int bit = 0;
    for (int right = side - 1; right >= 1; right -= 2) {
        if (right == 6) {
            right = 5; // skip the vertical timing column
        }
        for (int vertical = 0; vertical < side; vertical++) {
            for (int j = 0; j < 2; j++) {
                int x = right - j;
                bool isUpward = ((right + 1) & 2) == 0;
                int y = isUpward ? side - 1 - vertical : vertical;
                if (!isFunctionModule(work, x, y) && bit < count * 8) {
                    setModule(work, x, y, bitOf(codewords[bit >> 3], 7 - (bit & 7)));
                    bit++;
                }
            }
        }
    }
}

// ---- Masks ----

static bool maskInverts(int mask, int x, int y)
{
    switch (mask) {
    case 0:
        return (x + y) % 2 == 0;
    case 1:
        return y % 2 == 0;
    case 2:
        return x % 3 == 0;
    case 3:
        return (x + y) % 3 == 0;
    case 4:
        return (x / 3 + y / 2) % 2 == 0;
    case 5:
        return x * y % 2 + x * y % 3 == 0;
    case 6:
        return (x * y % 2 + x * y % 3) % 2 == 0;
    default:
        return ((x + y) % 2 + x * y % 3) % 2 == 0;
    }
}

// XOR, so applying the same mask twice restores the modules.
static void applyMask(Work *work, int mask)
{
    int side = work->code->side;
    for (int y = 0; y < side; y++) {
        for (int x = 0; x < side; x++) {
            if (!isFunctionModule(work, x, y) && maskInverts(mask, x, y)) {
                work->code->modules[y * side + x] ^= 1;
            }
        }
    }
}

// ---- Penalty (ISO/IEC 18004 7.8.3.1, Table 11) ----
// Read literally, as the segno library does: N1 adds 3 + (length - 5) per run
// of 5 or more, N2 3 per 2 x 2 block, N3 40 per 1:1:3:1:1 pattern with 4 light
// modules (or the symbol edge) before or after it, N4 10 per 5 % of imbalance.

static bool darkAt(const QrCode *code, int position, int line, bool isColumn)
{
    return isColumn ? code->isDark(line, position) : code->isDark(position, line);
}

static int runPenalty(const QrCode *code, int line, bool isColumn)
{
    int penalty = 0;
    int runLength = 0;
    bool isRunDark = false;
    for (int position = 0; position < code->side; position++) {
        bool isDark = darkAt(code, position, line, isColumn);
        if (position > 0 && isDark == isRunDark) {
            runLength++;
            continue;
        }
        if (runLength >= 5) {
            penalty += PENALTY_RUN + runLength - 5;
        }
        isRunDark = isDark;
        runLength = 1;
    }
    return runLength >= 5 ? penalty + PENALTY_RUN + runLength - 5 : penalty;
}

static bool hasFinderLikePatternAt(const QrCode *code, int line, bool isColumn, int start)
{
    static const bool PATTERN[7] = {true, false, true, true, true, false, true};
    for (int i = 0; i < 7; i++) {
        if (darkAt(code, start + i, line, isColumn) != PATTERN[i]) {
            return false;
        }
    }
    return true;
}

// True when [from, to) holds no dark module; the part outside the symbol counts as light.
static bool isLightSpan(const QrCode *code, int line, bool isColumn, int from, int to)
{
    for (int position = from < 0 ? 0 : from; position < to && position < code->side; position++) {
        if (darkAt(code, position, line, isColumn)) {
            return false;
        }
    }
    return true;
}

static int finderPenalty(const QrCode *code, int line, bool isColumn)
{
    int penalty = 0;
    int start = 0;
    while (start + 7 <= code->side) {
        if (!hasFinderLikePatternAt(code, line, isColumn, start)) {
            start++;
            continue;
        }
        bool hasLightBefore = isLightSpan(code, line, isColumn, start - 4, start);
        bool hasLightAfter = isLightSpan(code, line, isColumn, start + 7, start + 11);
        if (hasLightBefore || hasLightAfter) {
            penalty += PENALTY_FINDER;
            start += 7;
        } else {
            start += 4; // the pattern's last "dark light dark" may start the next one
        }
    }
    return penalty;
}

static int boxPenalty(const QrCode *code)
{
    int penalty = 0;
    for (int y = 0; y < code->side - 1; y++) {
        for (int x = 0; x < code->side - 1; x++) {
            bool isDark = code->isDark(x, y);
            if (isDark == code->isDark(x + 1, y) && isDark == code->isDark(x, y + 1) &&
                isDark == code->isDark(x + 1, y + 1)) {
                penalty += PENALTY_BOX;
            }
        }
    }
    return penalty;
}

static int balancePenalty(const QrCode *code)
{
    int total = code->side * code->side;
    int dark = 0;
    for (int i = 0; i < total; i++) {
        dark += code->modules[i];
    }
    double deviation = (double)dark / total * 100.0 - 50.0;
    if (deviation < 0) {
        deviation = -deviation;
    }
    return PENALTY_BALANCE * (int)(deviation / 5.0);
}

static int penaltyScore(const QrCode *code)
{
    int penalty = boxPenalty(code) + balancePenalty(code);
    for (int line = 0; line < code->side; line++) {
        penalty += runPenalty(code, line, false) + runPenalty(code, line, true);
        penalty += finderPenalty(code, line, false) + finderPenalty(code, line, true);
    }
    return penalty;
}

// Format and version information are added after the mask is chosen (ISO 7.8
// before 7.9/7.10), so their modules are light while masks are compared.
static void clearFormatAndVersionAreas(Work *work)
{
    int side = work->code->side;
    for (int i = 0; i <= 8; i++) {
        if (i != 6) { // the timing patterns cross here
            setModule(work, 8, i, false);
            setModule(work, i, 8, false);
        }
    }
    for (int i = 0; i < 8; i++) { // includes the dark module at (8, side - 8)
        setModule(work, 8, side - 1 - i, false);
        setModule(work, side - 1 - i, 8, false);
    }
    if (work->code->version < 7) {
        return;
    }
    for (int i = 0; i < 6; i++) {
        for (int j = side - 11; j < side - 8; j++) {
            setModule(work, j, i, false);
            setModule(work, i, j, false);
        }
    }
}

static int chooseMask(Work *work)
{
    clearFormatAndVersionAreas(work);
    int bestMask = 0;
    int bestPenalty = 0;
    for (int mask = 0; mask < MASK_COUNT; mask++) {
        applyMask(work, mask);
        int penalty = penaltyScore(work->code);
        if (mask == 0 || penalty < bestPenalty) {
            bestMask = mask;
            bestPenalty = penalty;
        }
        applyMask(work, mask);
    }
    return bestMask;
}

bool encode(const unsigned char *data, int length, QrCode *code, int forcedMask)
{
    if (!code || (!data && length > 0) || length < 0 || forcedMask < AUTOMATIC_MASK || forcedMask >= MASK_COUNT) {
        return false;
    }
    int version = smallestVersionFor(length);
    if (version == 0) {
        return false;
    }
    static Work work;
    memset(&work, 0, sizeof(work));
    work.code = code;
    code->version = version;
    code->side = version * 4 + 17;
    memset(code->modules, 0, sizeof(code->modules));

    static unsigned char dataBytes[MAX_CODEWORDS];
    static unsigned char allCodewords[MAX_CODEWORDS];
    buildDataCodewords(data, length, version, dataBytes);
    addEccAndInterleave(dataBytes, version, allCodewords);
    drawFunctionPatterns(&work);
    drawCodewords(&work, allCodewords, rawDataModules(version) / 8);

    code->mask = forcedMask == AUTOMATIC_MASK ? chooseMask(&work) : forcedMask;
    applyMask(&work, code->mask);
    drawFormatBits(&work, code->mask);
    drawVersionBits(&work);
    return true;
}

} // namespace QrEncoder
