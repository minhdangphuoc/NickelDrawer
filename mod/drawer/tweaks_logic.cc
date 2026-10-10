#include "tweaks_logic.h"

#include <string.h>

#include "drawer_logic.h"

namespace {

// Nesting allowed when skipping a value nobody reads (objects in arrays...).
const int MAX_SKIP_DEPTH = 16;

// A read position in the JSON text; failed is sticky.
struct Cursor {
    const char *at;
    const char *end;
    bool failed;
};

void fail(Cursor *cursor)
{
    cursor->failed = true;
}

void skipWhitespace(Cursor *cursor)
{
    while (cursor->at < cursor->end &&
           (*cursor->at == ' ' || *cursor->at == '\t' || *cursor->at == '\n' || *cursor->at == '\r')) {
        cursor->at++;
    }
}

// Consumes c (after whitespace) when it is next; true when it was.
bool consume(Cursor *cursor, char c)
{
    skipWhitespace(cursor);
    if (cursor->at < cursor->end && *cursor->at == c) {
        cursor->at++;
        return true;
    }
    return false;
}

bool expect(Cursor *cursor, char c)
{
    if (!consume(cursor, c)) {
        fail(cursor);
        return false;
    }
    return true;
}

char peek(Cursor *cursor)
{
    skipWhitespace(cursor);
    return cursor->at < cursor->end ? *cursor->at : '\0';
}

bool consumeWord(Cursor *cursor, const char *word)
{
    int length = (int)strlen(word);
    if (cursor->end - cursor->at < length || strncmp(cursor->at, word, length) != 0) {
        return false;
    }
    cursor->at += length;
    return true;
}

// ---- Strings ----

// Bounded UTF-8 output; a character that does not fit whole is dropped with
// everything after it, so the text never ends in half a character.
struct TextOut {
    char *buffer; // NULL: the text is only skipped
    int size;
    int length;
    bool isFull;
};

void appendBytes(TextOut *out, const char *bytes, int count)
{
    if (!out->buffer || out->isFull) {
        return;
    }
    if (out->length + count > out->size - 1) {
        out->isFull = true;
        return;
    }
    memcpy(out->buffer + out->length, bytes, count);
    out->length += count;
    out->buffer[out->length] = '\0';
}

void appendCodePoint(TextOut *out, unsigned codePoint)
{
    char bytes[4];
    int count;
    if (codePoint < 0x80) {
        bytes[0] = (char)codePoint;
        count = 1;
    } else if (codePoint < 0x800) {
        bytes[0] = (char)(0xC0 | (codePoint >> 6));
        bytes[1] = (char)(0x80 | (codePoint & 0x3F));
        count = 2;
    } else if (codePoint < 0x10000) {
        bytes[0] = (char)(0xE0 | (codePoint >> 12));
        bytes[1] = (char)(0x80 | ((codePoint >> 6) & 0x3F));
        bytes[2] = (char)(0x80 | (codePoint & 0x3F));
        count = 3;
    } else {
        bytes[0] = (char)(0xF0 | (codePoint >> 18));
        bytes[1] = (char)(0x80 | ((codePoint >> 12) & 0x3F));
        bytes[2] = (char)(0x80 | ((codePoint >> 6) & 0x3F));
        bytes[3] = (char)(0x80 | (codePoint & 0x3F));
        count = 4;
    }
    appendBytes(out, bytes, count);
}

// Length of the UTF-8 sequence starting with lead (1 for a stray byte, which
// is copied as is: appsctl writes valid UTF-8, Qt replaces anything else).
int utf8SequenceLength(unsigned char lead)
{
    if (lead >= 0xF0) {
        return 4;
    }
    if (lead >= 0xE0) {
        return 3;
    }
    return lead >= 0xC0 ? 2 : 1;
}

bool readHexDigits(Cursor *cursor, unsigned *value)
{
    if (cursor->end - cursor->at < 4) {
        return false;
    }
    *value = 0;
    for (int i = 0; i < 4; i++) {
        char c = *cursor->at++;
        unsigned digit;
        if (c >= '0' && c <= '9') {
            digit = c - '0';
        } else if (c >= 'a' && c <= 'f') {
            digit = c - 'a' + 10;
        } else if (c >= 'A' && c <= 'F') {
            digit = c - 'A' + 10;
        } else {
            return false;
        }
        *value = *value * 16 + digit;
    }
    return true;
}

// After "\u": one code point, joining a UTF-16 surrogate pair.
bool readUnicodeEscape(Cursor *cursor, unsigned *codePoint)
{
    unsigned first;
    if (!readHexDigits(cursor, &first)) {
        return false;
    }
    bool isHighSurrogate = first >= 0xD800 && first <= 0xDBFF;
    if (!isHighSurrogate) {
        *codePoint = (first >= 0xDC00 && first <= 0xDFFF) ? 0xFFFD : first;
        return true;
    }
    unsigned second;
    if (cursor->end - cursor->at < 2 || cursor->at[0] != '\\' || cursor->at[1] != 'u') {
        *codePoint = 0xFFFD;
        return true;
    }
    cursor->at += 2;
    if (!readHexDigits(cursor, &second) || second < 0xDC00 || second > 0xDFFF) {
        return false;
    }
    *codePoint = 0x10000 + ((first - 0xD800) << 10) + (second - 0xDC00);
    return true;
}

bool readEscape(Cursor *cursor, TextOut *out)
{
    if (cursor->at >= cursor->end) {
        return false;
    }
    char escaped = *cursor->at++;
    const char *simple = "\"\\/bfnrt";
    const char *meaning = "\"\\/\b\f\n\r\t";
    const char *found = strchr(simple, escaped);
    if (found && escaped) {
        appendBytes(out, meaning + (found - simple), 1);
        return true;
    }
    unsigned codePoint;
    if (escaped != 'u' || !readUnicodeEscape(cursor, &codePoint)) {
        return false;
    }
    appendCodePoint(out, codePoint);
    return true;
}

// A JSON string into buffer (NUL-terminated, at most size - 1 bytes); buffer
// NULL only skips it.
bool readString(Cursor *cursor, char *buffer, int size)
{
    TextOut out = {buffer, size, 0, false};
    if (buffer && size > 0) {
        buffer[0] = '\0';
    }
    if (!expect(cursor, '"')) {
        return false;
    }
    while (cursor->at < cursor->end) {
        unsigned char c = (unsigned char)*cursor->at;
        if (c == '"') {
            cursor->at++;
            return true;
        }
        if (c < 0x20) {
            break; // control characters must be escaped
        }
        if (c == '\\') {
            cursor->at++;
            if (!readEscape(cursor, &out)) {
                break;
            }
            continue;
        }
        int sequenceLength = utf8SequenceLength(c);
        if (cursor->end - cursor->at < sequenceLength) {
            break;
        }
        appendBytes(&out, cursor->at, sequenceLength);
        cursor->at += sequenceLength;
    }
    fail(cursor);
    return false;
}

// ---- Values ----

bool skipValue(Cursor *cursor, int depth);

bool skipContainer(Cursor *cursor, char close, bool hasKeys, int depth)
{
    if (depth > MAX_SKIP_DEPTH) {
        fail(cursor);
        return false;
    }
    cursor->at++; // the opening bracket
    if (consume(cursor, close)) {
        return true;
    }
    do {
        if (hasKeys && (!readString(cursor, NULL, 0) || !expect(cursor, ':'))) {
            return false;
        }
        if (!skipValue(cursor, depth + 1)) {
            return false;
        }
    } while (consume(cursor, ','));
    return expect(cursor, close);
}

bool skipNumber(Cursor *cursor)
{
    const char *start = cursor->at;
    while (cursor->at < cursor->end && strchr("+-0123456789.eE", *cursor->at) && *cursor->at) {
        cursor->at++;
    }
    if (cursor->at == start) {
        fail(cursor);
        return false;
    }
    return true;
}

bool skipValue(Cursor *cursor, int depth)
{
    char next = peek(cursor);
    if (next == '{') {
        return skipContainer(cursor, '}', true, depth);
    }
    if (next == '[') {
        return skipContainer(cursor, ']', false, depth);
    }
    if (next == '"') {
        return readString(cursor, NULL, 0);
    }
    if (consumeWord(cursor, "true") || consumeWord(cursor, "false") || consumeWord(cursor, "null")) {
        return true;
    }
    return skipNumber(cursor);
}

// true/false; anything else is skipped and reported as not a boolean.
bool readBoolean(Cursor *cursor, bool *value, bool *isBoolean)
{
    peek(cursor);
    *isBoolean = true;
    if (consumeWord(cursor, "true")) {
        *value = true;
        return true;
    }
    if (consumeWord(cursor, "false")) {
        *value = false;
        return true;
    }
    *isBoolean = false;
    return skipValue(cursor, 0);
}

// A whole number; anything else is skipped and reported as not a number.
bool readInteger(Cursor *cursor, int *value, bool *isNumber)
{
    *isNumber = false;
    if (peek(cursor) != '-' && (peek(cursor) < '0' || peek(cursor) > '9')) {
        return skipValue(cursor, 0);
    }
    bool isNegative = consume(cursor, '-');
    int number = 0;
    while (cursor->at < cursor->end && *cursor->at >= '0' && *cursor->at <= '9') {
        if (number < 100000) { // protocol versions are tiny; avoid overflow
            number = number * 10 + (*cursor->at - '0');
        }
        cursor->at++;
    }
    *value = isNegative ? -number : number;
    *isNumber = cursor->at >= cursor->end || !strchr(".eE", *cursor->at) || !*cursor->at;
    return *isNumber || skipNumber(cursor);
}

// A string value, or skips a value of any other type (buffer left empty).
bool readOptionalString(Cursor *cursor, char *buffer, int size)
{
    if (peek(cursor) == '"') {
        return readString(cursor, buffer, size);
    }
    buffer[0] = '\0';
    return skipValue(cursor, 0);
}

// Calls onKey for each key of an object; onKey reads (or skips) the value.
template <typename Target>
bool readObject(Cursor *cursor, Target *target, bool (*onKey)(Cursor *, const char *, Target *))
{
    if (!expect(cursor, '{')) {
        return false;
    }
    if (consume(cursor, '}')) {
        return true;
    }
    do {
        char key[32];
        if (!readString(cursor, key, sizeof(key)) || !expect(cursor, ':') || !onKey(cursor, key, target)) {
            fail(cursor);
            return false;
        }
    } while (consume(cursor, ','));
    return expect(cursor, '}');
}

// ---- Version ----

struct VersionField {
    bool hasVersion;
    int version;
};

bool readVersion(Cursor *cursor, VersionField *field)
{
    return readInteger(cursor, &field->version, &field->hasVersion);
}

TweaksLogic::ParseStatus versionStatus(const VersionField &field)
{
    switch (DrawerLogic::checkProtocolVersion(field.hasVersion, field.version)) {
    case DrawerLogic::PROTOCOL_OK:
        return TweaksLogic::PARSE_OK;
    case DrawerLogic::PROTOCOL_TOO_NEW:
        return TweaksLogic::PARSE_VERSION_TOO_NEW;
    case DrawerLogic::PROTOCOL_MISSING:
        break;
    }
    return TweaksLogic::PARSE_VERSION_MISSING;
}

// ---- tweaks ----

struct ParsedItem {
    TweaksLogic::Item item;
    char kind[16];
    bool hasKnownKind;
    bool valueIsBoolean;
};

bool onItemKey(Cursor *cursor, const char *key, ParsedItem *parsed)
{
    TweaksLogic::Item *item = &parsed->item;
    if (!strcmp(key, "id")) {
        return readOptionalString(cursor, item->id, sizeof(item->id));
    }
    if (!strcmp(key, "label")) {
        return readOptionalString(cursor, item->label, sizeof(item->label));
    }
    if (!strcmp(key, "kind")) {
        return readOptionalString(cursor, parsed->kind, sizeof(parsed->kind));
    }
    if (!strcmp(key, "confirm")) {
        return readOptionalString(cursor, item->confirm, sizeof(item->confirm));
    }
    if (!strcmp(key, "value")) {
        if (peek(cursor) == '"') {
            return readString(cursor, item->text, sizeof(item->text));
        }
        return readBoolean(cursor, &item->isOn, &parsed->valueIsBoolean);
    }
    return skipValue(cursor, 0);
}

bool kindFromName(const char *name, TweaksLogic::Kind *kind)
{
    static const struct {
        const char *name;
        TweaksLogic::Kind kind;
    } KINDS[] = {
        {"toggle", TweaksLogic::KIND_TOGGLE},
        {"info", TweaksLogic::KIND_INFO},
        {"action", TweaksLogic::KIND_ACTION},
    };
    for (unsigned i = 0; i < sizeof(KINDS) / sizeof(KINDS[0]); i++) {
        if (!strcmp(name, KINDS[i].name)) {
            *kind = KINDS[i].kind;
            return true;
        }
    }
    return false;
}

void keepItem(const ParsedItem &parsed, TweaksLogic::List *list)
{
    TweaksLogic::Kind kind;
    bool isUsable = parsed.item.id[0] && kindFromName(parsed.kind, &kind);
    if (!isUsable || list->count >= TweaksLogic::MAX_ITEMS) {
        list->skippedCount++; // a newer appsctl's kind, or junk: the rest still shows
        return;
    }
    TweaksLogic::Item *item = &list->items[list->count++];
    *item = parsed.item;
    item->kind = kind;
    if (kind == TweaksLogic::KIND_TOGGLE && !parsed.valueIsBoolean) {
        item->isOn = false; // spec: an unreadable setting is reported as false
    }
}

bool readItems(Cursor *cursor, TweaksLogic::List *list)
{
    if (!expect(cursor, '[')) {
        return false;
    }
    if (consume(cursor, ']')) {
        return true;
    }
    do {
        if (peek(cursor) != '{') {
            list->skippedCount++;
            if (!skipValue(cursor, 0)) {
                return false;
            }
            continue;
        }
        ParsedItem parsed;
        memset(&parsed, 0, sizeof(parsed));
        if (!readObject(cursor, &parsed, onItemKey)) {
            return false;
        }
        keepItem(parsed, list);
    } while (consume(cursor, ','));
    return expect(cursor, ']');
}

struct ParsedList {
    TweaksLogic::List *list;
    VersionField version;
    bool hasItems;
};

bool onListKey(Cursor *cursor, const char *key, ParsedList *parsed)
{
    if (!strcmp(key, "v")) {
        return readVersion(cursor, &parsed->version);
    }
    if (!strcmp(key, "items") && peek(cursor) == '[') {
        parsed->hasItems = true;
        return readItems(cursor, parsed->list);
    }
    return skipValue(cursor, 0);
}

// ---- tweak <id> ----

struct ParsedReply {
    TweaksLogic::Reply *reply;
    VersionField version;
    char nickel[32];
};

bool onErrorKey(Cursor *cursor, const char *key, TweaksLogic::Reply *reply)
{
    if (!strcmp(key, "message")) {
        return readOptionalString(cursor, reply->error, sizeof(reply->error));
    }
    return skipValue(cursor, 0);
}

bool readError(Cursor *cursor, TweaksLogic::Reply *reply)
{
    if (peek(cursor) == '{') {
        return readObject(cursor, reply, onErrorKey); // the protocol's {"code","message"}
    }
    return readOptionalString(cursor, reply->error, sizeof(reply->error));
}

bool onReplyKey(Cursor *cursor, const char *key, ParsedReply *parsed)
{
    TweaksLogic::Reply *reply = parsed->reply;
    bool isBoolean;
    if (!strcmp(key, "v")) {
        return readVersion(cursor, &parsed->version);
    }
    if (!strcmp(key, "ok")) {
        return readBoolean(cursor, &reply->isOk, &isBoolean);
    }
    if (!strcmp(key, "toast")) {
        return readOptionalString(cursor, reply->toast, sizeof(reply->toast));
    }
    if (!strcmp(key, "value")) {
        return readBoolean(cursor, &reply->value, &reply->hasValue);
    }
    if (!strcmp(key, "nickel")) {
        return readOptionalString(cursor, parsed->nickel, sizeof(parsed->nickel));
    }
    if (!strcmp(key, "error")) {
        return readError(cursor, reply);
    }
    return skipValue(cursor, 0);
}

TweaksLogic::NickelAction nickelActionFromName(const char *name)
{
    if (!name[0]) {
        return TweaksLogic::NICKEL_NONE;
    }
    if (!strcmp(name, "sleep")) {
        return TweaksLogic::NICKEL_SLEEP;
    }
    if (!strcmp(name, "reboot")) {
        return TweaksLogic::NICKEL_REBOOT;
    }
    return !strcmp(name, "rescan-library") ? TweaksLogic::NICKEL_RESCAN_LIBRARY : TweaksLogic::NICKEL_UNKNOWN;
}

// The whole text is one object and nothing follows it.
bool isWholeDocument(Cursor *cursor)
{
    skipWhitespace(cursor);
    return !cursor->failed && cursor->at == cursor->end;
}

} // namespace

namespace TweaksLogic {

ParseStatus parseList(const char *json, int length, List *list)
{
    memset(list, 0, sizeof(*list));
    if (!json || length <= 0) {
        return PARSE_BAD_JSON;
    }
    Cursor cursor = {json, json + length, false};
    ParsedList parsed = {list, {false, 0}, false};
    if (!readObject(&cursor, &parsed, onListKey) || !isWholeDocument(&cursor)) {
        memset(list, 0, sizeof(*list));
        return PARSE_BAD_JSON;
    }
    ParseStatus status = versionStatus(parsed.version);
    if (status != PARSE_OK) {
        return status;
    }
    return parsed.hasItems ? PARSE_OK : PARSE_NO_ITEMS;
}

ParseStatus parseReply(const char *json, int length, Reply *reply)
{
    memset(reply, 0, sizeof(*reply));
    if (!json || length <= 0) {
        return PARSE_BAD_JSON;
    }
    Cursor cursor = {json, json + length, false};
    ParsedReply parsed;
    memset(&parsed, 0, sizeof(parsed));
    parsed.reply = reply;
    if (!readObject(&cursor, &parsed, onReplyKey) || !isWholeDocument(&cursor)) {
        memset(reply, 0, sizeof(*reply));
        return PARSE_BAD_JSON;
    }
    reply->nickel = nickelActionFromName(parsed.nickel);
    return versionStatus(parsed.version);
}

int rowCount(int itemCount)
{
    return itemCount <= 0 ? 0 : (itemCount + COLUMN_COUNT - 1) / COLUMN_COUNT;
}

Rect cellRect(int indexOnPage, int gridWidth, int cellHeight, int dividerWidth)
{
    int row = indexOnPage / COLUMN_COUNT;
    int column = indexOnPage % COLUMN_COUNT;
    int leftWidth = (gridWidth - dividerWidth) / COLUMN_COUNT;
    int rightWidth = gridWidth - dividerWidth - leftWidth;
    Rect rect;
    rect.x = column == 0 ? 0 : leftWidth + dividerWidth;
    rect.y = row * (cellHeight + dividerWidth);
    rect.width = column == 0 ? leftWidth : rightWidth;
    rect.height = cellHeight;
    return rect;
}

int gridHeight(int itemCountOnPage, int cellHeight, int dividerWidth)
{
    int rows = rowCount(itemCountOnPage);
    return rows * (cellHeight + dividerWidth); // every row, the last too, ends with a divider
}

bool rowHasRightCell(int row, int itemCountOnPage)
{
    return row * COLUMN_COUNT + 1 < itemCountOnPage;
}

} // namespace TweaksLogic
