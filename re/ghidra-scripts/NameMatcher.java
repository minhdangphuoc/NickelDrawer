// Shared helper for the NickelDrawer scripts: find functions by regex over demangled and mangled names.
//@category NickelDrawer

import ghidra.program.model.listing.Function;
import ghidra.program.model.listing.FunctionIterator;
import ghidra.program.model.listing.Program;
import ghidra.program.model.symbol.Symbol;
import ghidra.util.task.TaskMonitor;

import java.util.ArrayList;
import java.util.List;
import java.util.regex.Pattern;

public class NameMatcher {

    private NameMatcher() {
    }

    public static List<Function> matchingFunctions(Program program, List<Pattern> patterns, TaskMonitor monitor) {
        List<Function> result = new ArrayList<>();
        FunctionIterator functions = program.getFunctionManager().getFunctions(true);
        while (functions.hasNext() && !monitor.isCancelled()) {
            Function function = functions.next();
            if (matchesAny(program, function, patterns)) {
                result.add(function);
            }
        }
        return result;
    }

    public static boolean matchesAny(Program program, Function function, List<Pattern> patterns) {
        List<String> names = namesOf(program, function);
        for (Pattern pattern : patterns) {
            for (String name : names) {
                if (pattern.matcher(name).find()) {
                    return true;
                }
            }
        }
        return false;
    }

    /** One line: address, qualified name with signature, every label (mangled names included). */
    public static String describe(Program program, Function function) {
        return function.getEntryPoint() + " " + String.join(" | ", namesOf(program, function));
    }

    private static List<String> namesOf(Program program, Function function) {
        List<String> names = new ArrayList<>();
        names.add(function.getName(true) + "(" + parameterTypes(function) + ")");
        for (Symbol symbol : program.getSymbolTable().getSymbols(function.getEntryPoint())) {
            String label = symbol.getName(true);
            if (!names.contains(label)) {
                names.add(label);
            }
        }
        return names;
    }

    private static String parameterTypes(Function function) {
        StringBuilder types = new StringBuilder();
        for (int i = 0; i < function.getParameterCount(); i++) {
            if (i > 0) {
                types.append(", ");
            }
            types.append(function.getParameter(i).getDataType().getDisplayName());
        }
        return types.toString();
    }
}
