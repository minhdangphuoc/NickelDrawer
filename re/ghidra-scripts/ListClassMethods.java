// Lists every function in the given C++ namespaces (classes), with address, parameter types and labels.
// Output holds only symbol names and addresses, no decompiled code.
//
// Usage (headless): -postScript ListClassMethods.java <outFile> <ClassName> [<ClassName> ...]
//@category NickelDrawer

import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.Function;

import java.io.File;
import java.io.PrintWriter;
import java.util.ArrayList;
import java.util.List;
import java.util.regex.Pattern;

public class ListClassMethods extends GhidraScript {

    @Override
    protected void run() throws Exception {
        String[] args = getScriptArgs();
        if (args.length < 2) {
            printerr("usage: ListClassMethods.java <outFile> <ClassName> [<ClassName> ...]");
            return;
        }
        File outFile = new File(args[0]);
        try (PrintWriter out = new PrintWriter(outFile, "UTF-8")) {
            for (int i = 1; i < args.length; i++) {
                writeClass(out, args[i]);
            }
        }
        println("ListClassMethods: wrote " + outFile);
    }

    private void writeClass(PrintWriter out, String className) {
        List<Pattern> patterns = new ArrayList<>();
        patterns.add(Pattern.compile("^" + Pattern.quote(className) + "::[^:]+\\("));
        out.println("== class " + className);
        for (Function function : NameMatcher.matchingFunctions(currentProgram, patterns, monitor)) {
            out.println("   " + NameMatcher.describe(currentProgram, function));
        }
    }
}
