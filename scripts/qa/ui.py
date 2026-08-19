# -*- coding: utf-8 -*-
"""أداةُ قيادةِ الشاشة نصّيّاً — لا صور."""
import subprocess, re, sys, time, io
sys.stdout.reconfigure(encoding="utf-8")
PKG = "com.rahalgo.customer"

def sh(*a, t=60):
    return subprocess.run(["adb"]+list(a), capture_output=True, timeout=t).stdout.decode("utf-8","replace")

def dump():
    """كلُّ ما على الشاشة: (النصّ، المركز، مفعَّل، قابلُ للضغط)."""
    for _ in range(3):
        sh("shell","uiautomator","dump","/sdcard/u.xml")
        x = sh("shell","cat","/sdcard/u.xml")
        if "<node" in x: break
        time.sleep(0.6)
    out=[]
    for n in re.finditer(r'<node[^>]*>', x):
        s=n.group(0)
        g=lambda k: (re.search(k+r'="([^"]*)"', s) or [None,""])[1]
        txt=g("text") or g("content-desc")
        b=re.search(r'bounds="\[(\d+),(\d+)\]\[(\d+),(\d+)\]"', s)
        if not b: continue
        x1,y1,x2,y2=map(int,b.groups())
        out.append({"t":txt,"c":((x1+x2)//2,(y1+y2)//2),"box":(x1,y1,x2,y2),
                    "en":g("enabled")=="true","clk":g("clickable")=="true",
                    "cls":g("class").split(".")[-1],"scroll":g("scrollable")=="true"})
    return out

def texts(nodes=None):
    return [n["t"] for n in (nodes or dump()) if n["t"].strip()]

def find(sub, nodes=None, clickable=False):
    for n in (nodes or dump()):
        if sub in n["t"] and (not clickable or n["clk"]):
            return n
    return None

def tap(sub, nodes=None):
    n=find(sub,nodes) or find(sub,nodes,False)
    if not n: raise LookupError("لم يُوجد: "+sub)
    sh("shell","input","tap",str(n["c"][0]),str(n["c"][1]))
    time.sleep(1.2); return n

def tapxy(x,y): sh("shell","input","tap",str(x),str(y)); time.sleep(1.2)
def typ(s): sh("shell","input","text",s.replace(" ","%s")); time.sleep(0.5)
def back(): sh("shell","input","keyevent","4"); time.sleep(1.0)
def swipe(x1,y1,x2,y2,ms=400): sh("shell","input","swipe",*map(str,(x1,y1,x2,y2,ms))); time.sleep(1.2)
def launch():
    sh("shell","monkey","-p",PKG,"-c","android.intent.category.LAUNCHER","1")
    time.sleep(3)
def crashes(since=None):
    a=["logcat","-d","-b","crash,main"]
    if since: a+=["-T",since]
    x=sh(*a)
    # **و٤٠١ ملتقَطةٌ ليست انهياراً** — الانهيارُ ما يُسقط العمليّة.
    return [l for l in x.splitlines()
            if re.search(r"FATAL EXCEPTION|ANR in |E AndroidRuntime|Process.*has died", l)
            and "not_signed_in" not in l]
def show(label, nodes=None):
    print("┌─", label)
    for t in texts(nodes): print("│ ", t)
    print("└─")

def hit(sub, nodes=None, idx=0):
    """**يُضغط أصغرُ عنصرٍ قابلٍ للضغط يحتوي النصّ** — لا مركزُ النصّ نفسِه.

    (وُجد ٢٠٢٦-٠٨-١٩: نصُّ «−» داخلَ زرٍّ أوسعَ منه، فمركزُ النصّ
    يقع أحياناً خارجَ منطقة اللمس.)"""
    n = nodes or dump()
    tgt = [x for x in n if sub in x["t"]]
    if not tgt: raise LookupError("لا نصّ: "+sub)
    t = tgt[idx]
    x1,y1,x2,y2 = t["box"]
    best = None
    for c in n:
        if not c["clk"]: continue
        a,b,d,e = c["box"]
        if a<=x1 and b<=y1 and d>=x2 and e>=y2:
            area = (d-a)*(e-b)
            if best is None or area < best[0]: best = (area, c)
    node = best[1] if best else t
    sh("shell","input","tap",str(node["c"][0]),str(node["c"][1]))
    time.sleep(1.5)
    return node

def total():
    t = texts()
    for i,x in enumerate(t):
        if x == "المجموع" and i+1 < len(t): return t[i+1]
    return None
