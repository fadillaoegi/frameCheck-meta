# Isometric two-plate mark: build frame over design plate, registration point seen through the window.
def P(pts): return " ".join(f"{x:.2f},{y:.2f}" for x,y in pts)
def rh(cx,cy,w,h): return [(cx,cy-h),(cx+w,cy),(cx,cy+h),(cx-w,cy)]
def plate(cx,cy,w,h,t,top,left,right):
    T=rh(cx,cy,w,h)
    L=[T[3],T[2],(T[2][0],T[2][1]+t),(T[3][0],T[3][1]+t)]
    R=[T[2],T[1],(T[1][0],T[1][1]+t),(T[2][0],T[2][1]+t)]
    return f'<polygon points="{P(L)}" fill="{left}"/><polygon points="{P(R)}" fill="{right}"/><polygon points="{P(T)}" fill="{top}"/>'
def path(pts): return "M"+" L".join(f"{x:.2f},{y:.2f}" for x,y in pts)+" Z"

W,H=24,12
dc=(32,35); fc=(32,25); t=3.5
iw,ih=12.5,6.25
PAPER = dict(ink="#1b1915", inkL="#3a362e", inkR="#0e0d0a", blue="#2b4acb", blueL="#1f369a", blueR="#162a7a",
             red="#d4412a", cross="#f3efe6", window="#f3efe6", edge=None)
# Dark UI variant: cyan design plate, a dark frame with a magenta neon edge, a hot-pink finding.
NEON = dict(ink="#191930", inkL="#2a2a4a", inkR="#0f0f1f", blue="#00d4f0", blueL="#0098b3", blueR="#006b80",
            red="#ff2e63", cross="#07070e", window="#07070e", edge="#ff2bd6")
pal = PAPER

def mark(bg=None, p=None):
    global pal
    pal = p or PAPER
    ink, inkL, inkR, blue, blueL, blueR, red = (pal[k] for k in ("ink", "inkL", "inkR", "blue", "blueL", "blueR", "red"))
    s=[]
    if bg: s.append(f'<rect x="4" y="1" width="56" height="56" rx="12" fill="{bg}"/>')
    # design plate
    s.append(plate(*dc,W,H,t,blue,blueL,blueR))
    # registration cross on design plate, along iso axes, under the window
    cx,cy=32,29.2
    s.append('<defs><clipPath id="win"><polygon points="%s"/></clipPath></defs>'%P(rh(fc[0],fc[1],iw,ih)))
    # build frame: outer sides then ring top with hole (evenodd)
    O=rh(*fc,W,H); I=rh(*fc,iw,ih)
    L=[O[3],O[2],(O[2][0],O[2][1]+t),(O[3][0],O[3][1]+t)]
    R=[O[2],O[1],(O[1][0],O[1][1]+t),(O[2][0],O[2][1]+t)]
    s.append(f'<polygon points="{P(L)}" fill="{inkL}"/><polygon points="{P(R)}" fill="{inkR}"/>')
    # what is seen through the window: background, then design plate, then the cross & dot, then inner far walls
    g=['<g clip-path="url(#win)">']
    g.append(f'<polygon points="{P(rh(*fc,iw,ih))}" fill="{bg or pal['window']}"/>')
    g.append(plate(*dc,W,H,t,blue,blueL,blueR))
    g.append(f'<path d="M{cx-9},{cy-4.5} L{cx+9},{cy+4.5} M{cx+9},{cy-4.5} L{cx-9},{cy+4.5}" stroke="{pal['cross']}" stroke-width="0.9" opacity="0.85"/>')
    g.append(f'<ellipse cx="{cx}" cy="{cy}" rx="4.6" ry="2.3" fill="none" stroke="{pal['cross']}" stroke-width="0.9" opacity="0.85"/>')
    g.append(f'<ellipse cx="{cx}" cy="{cy}" rx="2.2" ry="1.1" fill="{red}"/>')
    # inner far walls (back-left and back-right), dropping t from the hole's top edges
    BL=[I[3],I[0],(I[0][0],I[0][1]+t),(I[3][0],I[3][1]+t)]
    BR=[I[0],I[1],(I[1][0],I[1][1]+t),(I[0][0],I[0][1]+t)]
    g.append(f'<polygon points="{P(BL)}" fill="{inkR}"/><polygon points="{P(BR)}" fill="{inkL}"/>')
    g.append('</g>')
    s.append("".join(g))
    s.append(f'<path d="{path(O)} {path(I)}" fill="{ink}" fill-rule="evenodd"/>')
    if pal["edge"]:
        s.append(f'<path d="{path(O)} {path(I)}" fill="none" stroke="{pal["edge"]}" stroke-width="0.7" stroke-linejoin="round"/>')
    # evidence post: from the registration point up through the frame
    s.append(f'<path d="M{cx},{cy-1.2} L{cx},6" stroke="{red}" stroke-width="1.2" stroke-dasharray="2 1.6" stroke-linecap="round"/>')
    s.append(f'<circle cx="{cx}" cy="5" r="1.8" fill="{red}"/>')
    return "".join(s)

def svg(inner,vb="4 1 56 56"): return f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{vb}">{inner}</svg>'
open("framecheck-mark.svg","w").write(svg(mark()))
open("framecheck-app-icon.svg","w").write(svg(mark("#f3efe6")))
open("framecheck-mark-neon.svg","w").write(svg(mark(None, NEON)))
open("framecheck-app-icon-neon.svg","w").write(svg(mark("#07070e", NEON)))
