import { readFile, stat } from "node:fs/promises";
import { fileURLToPath, pathToFileURL } from "node:url";
import { join, dirname } from "node:path";
const WEB = join(dirname(fileURLToPath(import.meta.url)), "..");
async function exists(p) { try { await stat(p); return true; } catch { return false; } }
export async function resolve(spec, context, next) {
  let s = spec;
  if (s.startsWith("@/")) s = pathToFileURL(join(WEB, "apps", "rahalgo", "src", s.slice(2))).href;
  if ((s.startsWith("./") || s.startsWith("../") || s.startsWith("file:")) && !/\.(ts|tsx|json|mjs|js)$/.test(s)) {
    const base = s.startsWith("file:") ? s : new URL(s, context.parentURL).href;
    for (const ext of [".ts", ".tsx", "/index.ts"]) {
      if (await exists(fileURLToPath(base + ext))) return next(base + ext, context);
    }
  }
  return next(s, context);
}
export async function load(url, context, next) {
  // **والامتدادُ يُعلَن نوعاً** — وإلّا حذّر node من حزمةٍ بلا `type`.
  if (/\.(ts|tsx)$/.test(url)) return next(url, { ...context, format: "module-typescript" });
  if (url.endsWith(".json")) {
    const src = await readFile(fileURLToPath(url), "utf8");
    return { format: "module", shortCircuit: true, source: "export default " + src };
  }
  return next(url, context);
}
