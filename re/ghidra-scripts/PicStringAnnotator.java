// Resolves ARM position-independent string loads in decompiler output.
// GCC emits "literal-pool word + pc"; Ghidra prints that as "DAT_<pool> + 0x<pc>". This reads the pool word,
// adds the pc and, if the target is a NUL-terminated ASCII string, appends it as a comment.
//@category NickelDrawer

import ghidra.program.model.address.Address;
import ghidra.program.model.listing.Program;
import ghidra.program.model.mem.Memory;
import ghidra.program.model.mem.MemoryAccessException;

import java.util.regex.Matcher;
import java.util.regex.Pattern;

public class PicStringAnnotator {

    private static final Pattern PIC_LOAD = Pattern.compile("DAT_([0-9a-f]{8}) \\+ 0x([0-9a-f]+)");
    private static final int MAX_STRING_LENGTH = 200;

    private PicStringAnnotator() {
    }

    public static String annotate(Program program, String code) {
        StringBuilder annotated = new StringBuilder();
        for (String line : code.split("\n", -1)) {
            annotated.append(line);
            Matcher matcher = PIC_LOAD.matcher(line);
            StringBuilder comment = new StringBuilder();
            while (matcher.find()) {
                String text = resolve(program, matcher.group(1), matcher.group(2));
                if (text != null) {
                    comment.append(" \"").append(text).append("\"");
                }
            }
            if (comment.length() > 0) {
                annotated.append("  /*").append(comment).append(" */");
            }
            annotated.append("\n");
        }
        return annotated.toString();
    }

    private static String resolve(Program program, String poolHex, String pcHex) {
        try {
            Memory memory = program.getMemory();
            Address pool = program.getAddressFactory().getDefaultAddressSpace().getAddress(Long.parseLong(poolHex, 16));
            long offset = memory.getInt(pool) & 0xffffffffL;
            long target = (offset + Long.parseLong(pcHex, 16)) & 0xffffffffL;
            return readCString(memory, pool.getNewAddress(target));
        } catch (MemoryAccessException | RuntimeException e) {
            return null;
        }
    }

    private static String readCString(Memory memory, Address start) throws MemoryAccessException {
        StringBuilder text = new StringBuilder();
        for (int i = 0; i < MAX_STRING_LENGTH; i++) {
            byte value = memory.getByte(start.add(i));
            if (value == 0) {
                return text.length() == 0 ? null : text.toString();
            }
            if (value < 0x20 || value > 0x7e) {
                return null;
            }
            text.append((char) value);
        }
        return null;
    }
}
