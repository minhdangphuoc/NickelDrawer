#ifndef NICKELDRAWER_TEST_EXPECT_H
#define NICKELDRAWER_TEST_EXPECT_H

// Minimal assertions for the host unit tests (no C++ standard library).

#include <stdio.h>
#include <string.h>

static int failures = 0;

#define EXPECT_EQ(expected, actual)                                                                      \
    do {                                                                                                 \
        long expectedValue = (long)(expected);                                                           \
        long actualValue = (long)(actual);                                                               \
        if (expectedValue != actualValue) {                                                              \
            printf("FAIL %s:%d: %s == %ld, expected %ld\n", __FILE__, __LINE__, #actual, actualValue,     \
                   expectedValue);                                                                       \
            failures++;                                                                                  \
        }                                                                                                \
    } while (0)

#define EXPECT_STREQ(expected, actual)                                                                   \
    do {                                                                                                 \
        const char *expectedText = (expected);                                                           \
        const char *actualText = (actual);                                                               \
        if (strcmp(expectedText, actualText) != 0) {                                                     \
            printf("FAIL %s:%d: %s == \"%s\", expected \"%s\"\n", __FILE__, __LINE__, #actual, actualText, \
                   expectedText);                                                                        \
            failures++;                                                                                  \
        }                                                                                                \
    } while (0)

static int finishTests(const char *suite)
{
    if (failures) {
        printf("%s: %d failure(s)\n", suite, failures);
        return 1;
    }
    printf("%s: all tests passed\n", suite);
    return 0;
}

#endif
