// Decompiles every function whose qualified (demangled) or mangled name matches one of the regexes,
// writing C pseudocode to a file. Output is Kobo's copyrighted code: keep it under ~/.cache/nickeldrawer/decomp/.
//
// Usage (headless): -postScript DecompileByName.java <outFile> <regex> [<regex> ...]
// Regexes use java.util.regex find() semantics against "Namespace::name(signature)" and every label at the entry.
//@category NickelDrawer

import ghidra.app.decompiler.DecompInterface;
import ghidra.app.decompiler.DecompileResults;
import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.Function;

import java.io.File;
import java.io.PrintWriter;
import java.util.ArrayList;
import java.util.List;
import java.util.regex.Pattern;

public class DecompileByName extends GhidraScript {

    private static final int DECOMPILE_TIMEOUT_SECONDS = 120;

    @Override
    protected void run() throws Exception {
        String[] args = getScriptArgs();
        if (args.length < 2) {
            printerr("usage: DecompileByName.java <outFile> <regex> [<regex> ...]");
            return;
        }
        File outFile = new File(args[0]);
        List<Pattern> patterns = new ArrayList<>();
        for (int i = 1; i < args.length; i++) {
            patterns.add(Pattern.compile(args[i]));
        }

        DecompInterface decompiler = new DecompInterface();
        if (!decompiler.openProgram(currentProgram)) {
            printerr("decompiler failed to open program: " + decompiler.getLastMessage());
            return;
        }
        int matched = 0;
        try (PrintWriter out = new PrintWriter(outFile, "UTF-8")) {
            for (Function function : NameMatcher.matchingFunctions(currentProgram, patterns, monitor)) {
                matched++;
                writeDecompiled(out, decompiler, function);
            }
        } finally {
            decompiler.dispose();
        }
        println("DecompileByName: " + matched + " functions -> " + outFile);
    }

    private void writeDecompiled(PrintWriter out, DecompInterface decompiler, Function function) {
        out.println("//// " + NameMatcher.describe(currentProgram, function));
        DecompileResults results = decompiler.decompileFunction(function, DECOMPILE_TIMEOUT_SECONDS, monitor);
        if (results == null || !results.decompileCompleted()) {
            out.println("// decompile failed: " + (results == null ? "null" : results.getErrorMessage()));
        } else {
            out.println(PicStringAnnotator.annotate(currentProgram, results.getDecompiledFunction().getC()));
        }
        out.println();
    }
}
