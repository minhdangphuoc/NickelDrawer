// Prints the slots of C++ vtables (Itanium ABI, 32-bit ARM): slot index, byte offset from the address point,
// and the name of the function each slot points to. Output holds only symbol names and offsets.
//
// Usage (headless): -postScript DumpVtable.java <outFile> <mangledVtableSymbol> [<maxSlots>] [<sym> <maxSlots> ...]
// Example: _ZTV18AbstractController 16
//@category NickelDrawer

import ghidra.app.script.GhidraScript;
import ghidra.program.model.address.Address;
import ghidra.program.model.listing.Function;
import ghidra.program.model.symbol.Symbol;

import java.io.File;
import java.io.PrintWriter;

public class DumpVtable extends GhidraScript {

    private static final int POINTER_SIZE = 4;
    private static final int ADDRESS_POINT_OFFSET = 2 * POINTER_SIZE; // offset-to-top + typeinfo precede slot 0

    @Override
    protected void run() throws Exception {
        String[] args = getScriptArgs();
        if (args.length < 2) {
            printerr("usage: DumpVtable.java <outFile> <vtableSymbol> [<maxSlots>] ...");
            return;
        }
        try (PrintWriter out = new PrintWriter(new File(args[0]), "UTF-8")) {
            int index = 1;
            while (index < args.length) {
                String symbolName = args[index++];
                int maxSlots = 40;
                if (index < args.length && args[index].matches("\\d+")) {
                    maxSlots = Integer.parseInt(args[index++]);
                }
                writeVtable(out, symbolName, maxSlots);
            }
        }
        println("DumpVtable: wrote " + args[0]);
    }

    private void writeVtable(PrintWriter out, String symbolName, int maxSlots) throws Exception {
        out.println("== " + symbolName);
        Symbol symbol = findSymbol(symbolName);
        if (symbol == null) {
            out.println("   not found");
            return;
        }
        Address addressPoint = symbol.getAddress().add(ADDRESS_POINT_OFFSET);
        for (int slot = 0; slot < maxSlots; slot++) {
            Address slotAddress = addressPoint.add((long) slot * POINTER_SIZE);
            long target = getInt(slotAddress) & 0xffffffffL;
            if (target == 0 && slot > 0) {
                out.println(String.format("   [%2d] +0x%02x  0 (end or offset-to-top of a secondary vtable)", slot, slot * POINTER_SIZE));
                break;
            }
            out.println(String.format("   [%2d] +0x%02x  %s", slot, slot * POINTER_SIZE, describeTarget(target)));
        }
    }

    private Symbol findSymbol(String name) {
        for (Symbol symbol : currentProgram.getSymbolTable().getSymbols(name)) {
            return symbol;
        }
        return null;
    }

    private String describeTarget(long target) {
        Address address = toAddr(target & ~1L); // Thumb bit
        Function function = getFunctionAt(address);
        if (function != null) {
            return address + " " + function.getName(true);
        }
        Symbol symbol = getSymbolAt(address);
        return address + " " + (symbol == null ? "<no symbol>" : symbol.getName(true));
    }
}
