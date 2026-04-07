package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashHTML))
}

const dashHTML = `<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>Strongbox</title>
<style>:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--mono:'JetBrains Mono',monospace}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--mono);line-height:1.5}
.hdr{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}.hdr h1{font-size:.9rem;letter-spacing:2px}
.main{padding:1.5rem;max-width:900px;margin:0 auto}
.env-bar{display:flex;gap:.3rem;margin-bottom:1rem}
.env-btn{font-size:.6rem;padding:.25rem .6rem;border:1px solid var(--bg3);background:var(--bg);color:var(--cm);cursor:pointer}.env-btn:hover{border-color:var(--leather)}.env-btn.active{border-color:var(--gold);color:var(--gold)}
.secret{background:var(--bg2);border:1px solid var(--bg3);padding:.7rem 1rem;margin-bottom:.4rem;display:flex;justify-content:space-between;align-items:center}
.secret-name{font-size:.8rem;color:var(--cream)}
.secret-meta{font-size:.6rem;color:var(--cm);margin-top:.1rem}
.secret-val{font-family:var(--mono);font-size:.7rem;color:var(--cm);background:var(--bg);padding:.2rem .4rem;border:1px solid var(--bg3);cursor:pointer;max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.secret-val:hover{color:var(--cream)}
.btn{font-size:.6rem;padding:.2rem .5rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd)}.btn:hover{border-color:var(--leather);color:var(--cream)}
.btn-p{background:var(--rust);border-color:var(--rust);color:var(--bg)}
.modal-bg{display:none;position:fixed;inset:0;background:rgba(0,0,0,.6);z-index:100;align-items:center;justify-content:center}.modal-bg.open{display:flex}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:400px;max-width:90vw}
.modal h2{font-size:.8rem;margin-bottom:1rem;color:var(--rust)}
.fr{margin-bottom:.5rem}.fr label{display:block;font-size:.55rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-bottom:.15rem}
.fr input,.fr select,.fr textarea{width:100%;padding:.35rem .5rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.acts{display:flex;gap:.4rem;justify-content:flex-end;margin-top:.8rem}
.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic;font-size:.75rem}
</style></head><body>
<div class="hdr"><h1>STRONGBOX</h1><button class="btn btn-p" onclick="openForm()">+ Add Secret</button></div>
<div class="main">
<div class="env-bar" id="envs"></div>
<div id="secrets"></div>
</div>
<div class="modal-bg" id="mbg" onclick="if(event.target===this)cm()"><div class="modal" id="mdl"></div></div>
<script>
const A='/api';let secrets=[],filterEnv='';
async function load(){const r=await fetch(A+'/secrets').then(r=>r.json());secrets=r.secrets||[];
const envs=[...new Set(secrets.map(s=>s.environment).filter(e=>e))];
let h='<button class="env-btn'+(filterEnv===''?' active':'')+'" onclick="setEnv(\'\')">All ('+secrets.length+')</button>';
envs.forEach(e=>{const c=secrets.filter(s=>s.environment===e).length;h+='<button class="env-btn'+(filterEnv===e?' active':'')+'" onclick="setEnv(\''+e+'\')">'+esc(e)+' ('+c+')</button>';});
document.getElementById('envs').innerHTML=h;render();}
function setEnv(e){filterEnv=e;load();}
function render(){let filtered=filterEnv?secrets.filter(s=>s.environment===filterEnv):secrets;
if(!filtered.length){document.getElementById('secrets').innerHTML='<div class="empty">No secrets stored.</div>';return;}
let h='';filtered.forEach(s=>{
const masked='•'.repeat(Math.min(s.value.length,20))||'(empty)';
h+='<div class="secret"><div><div class="secret-name">'+esc(s.name)+'</div><div class="secret-meta">v'+s.version+' · '+esc(s.environment)+(s.description?' · '+esc(s.description):'')+'</div></div><div style="display:flex;gap:.4rem;align-items:center"><span class="secret-val" onclick="reveal(this,\''+esc(s.value)+'\')">'+masked+'</span><button class="btn" onclick="del(\''+s.id+'\')" style="color:var(--cm)">✕</button></div></div>';});
document.getElementById('secrets').innerHTML=h;}
function reveal(el,val){if(el.dataset.revealed){el.textContent='•'.repeat(val.length);el.dataset.revealed=''}else{el.textContent=val;el.dataset.revealed='1'}}
async function del(id){if(confirm('Delete?')){await fetch(A+'/secrets/'+id,{method:'DELETE'});load();}}
function openForm(){document.getElementById('mdl').innerHTML='<h2>Add Secret</h2><div class="fr"><label>Name</label><input id="f-n" placeholder="e.g. DATABASE_URL"></div><div class="fr"><label>Value</label><textarea id="f-v" rows="3" placeholder="secret value"></textarea></div><div class="fr"><label>Environment</label><input id="f-e" value="default" placeholder="default, staging, production"></div><div class="fr"><label>Description</label><input id="f-d" placeholder="optional note"></div><div class="acts"><button class="btn" onclick="cm()">Cancel</button><button class="btn btn-p" onclick="sub()">Store</button></div>';document.getElementById('mbg').classList.add('open');}
async function sub(){await fetch(A+'/secrets',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:document.getElementById('f-n').value,value:document.getElementById('f-v').value,environment:document.getElementById('f-e').value,description:document.getElementById('f-d').value})});cm();load();}
function cm(){document.getElementById('mbg').classList.remove('open');}
function esc(s){if(!s)return'';const d=document.createElement('div');d.textContent=s;return d.innerHTML;}
load();
</script><script>
(function(){
  fetch('/api/config').then(function(r){return r.json()}).then(function(cfg){
    if(!cfg||typeof cfg!=='object')return;
    if(cfg.dashboard_title){
      document.title=cfg.dashboard_title;
      var h1=document.querySelector('h1');
      if(h1){
        var inner=h1.innerHTML;
        var firstSpan=inner.match(/<span[^>]*>[^<]*<\/span>/);
        if(firstSpan){h1.innerHTML=firstSpan[0]+' '+cfg.dashboard_title}
        else{h1.textContent=cfg.dashboard_title}
      }
    }
  }).catch(function(){});
})();
</script>
</body></html>`
