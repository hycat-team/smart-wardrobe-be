// One-off fixer for feature 017: reattach swagger doc-comment blocks that
// were separated from their functions during the handler file splits.
// A dangling block is a comment group containing "@Router" that is NOT
// immediately followed by a func declaration. Each block's first line names
// its function ("// <FuncName> ..."), which is used to locate the target
// declaration in the same package.
import fs from "node:fs";

const dir = "internal/modules/brand/presentation/handler/";
const files = fs.readdirSync(dir).filter((f) => f.endsWith(".go") && !f.endsWith("_test.go"));

// 1. Extract dangling comment blocks.
const dangling = [];
for (const f of files) {
  const lines = fs.readFileSync(dir + f, "utf8").split("\n");
  for (let i = 0; i < lines.length; i++) {
    if (!lines[i].trimStart().startsWith("//")) continue;
    if (!lines[i].includes("@Router")) continue;
    // walk up over contiguous comment lines
    let begin = i;
    while (begin > 0 && lines[begin - 1].trimStart().startsWith("//")) begin--;
    const next = (lines[i + 1] || "").trim();
    if (next.startsWith("func ")) continue; // properly attached
    const first = lines[begin].trimStart().replace(/^\/\/\s*/, "");
    const funcName = first.split(/\s+/)[0];
    dangling.push({ file: f, begin, end: i, funcName, block: lines.slice(begin, i + 1) });
    i = i + 1; // skip ahead
  }
}

console.log("dangling blocks:", dangling.map((d) => `${d.funcName} @ ${d.file}:${d.begin + 1}`).join(" | "));

// 2. Remove dangling blocks from their files (bottom-up per file).
const byFile = new Map();
for (const d of dangling) {
  if (!byFile.has(d.file)) byFile.set(d.file, []);
  byFile.get(d.file).push(d);
}
for (const [f, ds] of byFile) {
  const lines = fs.readFileSync(dir + f, "utf8").split("\n");
  for (const d of ds.sort((a, b) => b.begin - a.begin)) {
    lines.splice(d.begin, d.end - d.begin + 1);
  }
  fs.writeFileSync(dir + f, lines.join("\n"));
}

// 3. Insert each block above its function declaration in the package.
for (const d of dangling) {
  let target = null;
  for (const f of files) {
    if (f === d.file) continue;
    const lines = fs.readFileSync(dir + f, "utf8").split("\n");
    const idx = lines.findIndex((l) =>
      new RegExp(`^func \\((h|uc|s) \\*\\w+\\) ${d.funcName}\\(`).test(l) ||
      new RegExp(`^func ${d.funcName}\\(`).test(l),
    );
    if (idx >= 0) {
      target = { f, lines, idx };
      break;
    }
  }
  if (!target) {
    console.log("NO TARGET for", d.funcName);
    continue;
  }
  const { f, lines, idx } = target;
  // ensure a blank line between inserted block and func? gofmt keeps comment adjacent to func.
  lines.splice(idx, 0, ...d.block);
  fs.writeFileSync(dir + f, lines.join("\n"));
  console.log(`reattached ${d.funcName} -> ${f}:${idx + 1}`);
}
