// Host unit tests for the locale pick the helper's strings use (mod/common/locale_chain.cc, no Qt, no
// C++ standard library). Run: make -C helpers/extras docker-test

#include "../../../mod/common/test_expect.h"
#include "../../../mod/common/locale_chain.h"

using namespace LocaleChain;

static char names[3][LOCALE_NAME_SIZE];

static void testExactThenLanguageThenEnglish()
{
    EXPECT_EQ(3, fileNames("pt_BR", names, 3));
    EXPECT_STREQ("pt_BR", names[0]);
    EXPECT_STREQ("pt", names[1]);
    EXPECT_STREQ("en", names[2]);
}

static void testVietnameseAndEnglishDeviceLocales()
{
    EXPECT_EQ(3, fileNames("vi_VN", names, 3));
    EXPECT_STREQ("vi_VN", names[0]);
    EXPECT_STREQ("vi", names[1]);
    EXPECT_EQ(2, fileNames("vi", names, 3));
    EXPECT_STREQ("vi", names[0]);
    EXPECT_STREQ("en", names[1]);
    EXPECT_EQ(1, fileNames("en", names, 3)); // English is not listed twice
    EXPECT_STREQ("en", names[0]);
}

static void testMissingOrUnsafeLocalesFallBackToEnglish()
{
    EXPECT_EQ(1, fileNames("", names, 3));
    EXPECT_STREQ("en", names[0]);
    EXPECT_EQ(1, fileNames(NULL, names, 3));
    EXPECT_EQ(1, fileNames("../../etc/passwd", names, 3)); // never becomes a path
    EXPECT_STREQ("en", names[0]);
    EXPECT_EQ(1, fileNames("a_very_long_locale_name", names, 3));
}

static void testTheListIsCutAtTheCallersCapacity()
{
    EXPECT_EQ(2, fileNames("pt_BR", names, 2));
    EXPECT_STREQ("pt_BR", names[0]);
    EXPECT_STREQ("pt", names[1]);
}

int main()
{
    testExactThenLanguageThenEnglish();
    testVietnameseAndEnglishDeviceLocales();
    testMissingOrUnsafeLocalesFallBackToEnglish();
    testTheListIsCutAtTheCallersCapacity();
    return finishTests("locale_chain");
}
