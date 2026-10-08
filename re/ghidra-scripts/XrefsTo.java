// Lists the callers (and other references) of every function whose name matches one of the regexes.
// Output holds only addresses and symbol names, no decompiled code.
//
// Usage (headless): -postScript XrefsTo.java <outFile> <regex> [<regex> ...]
//@category NickelDrawer

import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.Function;
import ghidra.program.model.symbol.Reference;
import ghidra.program.model.symbol.ReferenceIterator;

import java.io.File;
import java.io.PrintWriter;
import java.util.ArrayList;
import java.util.List;
import java.util.regex.Pattern;

public class XrefsTo extends GhidraScript {

    @Override
    protected void run() throws Exception {
        String[] args = getScriptArgs();
        if (args.length < 2) {
            printerr("usage: XrefsTo.java <outFile> <regex> [<regex> ...]");
            return;
        }
        File outFile = new File(args[0]);
        List<Pattern> patterns = new ArrayList<>();
        for (int i = 1; i < args.length; i++) {
            patterns.add(Pattern.compile(args[i]));
        }
        try (PrintWriter out = new PrintWriter(outFile, "UTF-8")) {
            for (Function target : NameMatcher.matchingFunctions(currentProgram, patterns, monitor)) {
                out.println("== " + NameMatcher.describe(currentProgram, target));
                writeReferences(out, target);
            }
        }
        println("XrefsTo: wrote " + outFile);
    }

    private void writeReferences(PrintWriter out, Function target) {
        ReferenceIterator references = currentProgram.getReferenceManager().getReferencesTo(target.getEntryPoint());
        while (references.hasNext()) {
            Reference reference = references.next();
            Function caller = getFunctionContaining(reference.getFromAddress());
            String callerName = caller == null ? "<no function>" : caller.getName(true);
            out.println("   " + reference.getReferenceType() + " from " + reference.getFromAddress() + " in " + callerName);
        }
    }
}
