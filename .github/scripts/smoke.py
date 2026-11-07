import json,os,re,subprocess,sys,tempfile,threading,queue,urllib.request,urllib.parse
from pathlib import Path

repo,executable=sys.argv[1:3]
def multipart(fields):
 boundary='portfolio-smoke-boundary'
 data=b''
 for name,value in fields.items():
  data+=(f'--{boundary}\r\nContent-Disposition: form-data; name="{name}"\r\n\r\n{value}\r\n').encode()
 data+=f'--{boundary}--\r\n'.encode()
 return data,'multipart/form-data; boundary='+boundary

def launch(env):
 p=subprocess.Popen([str(Path(executable).resolve())],env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,encoding='utf-8')
 q=queue.Queue()
 threading.Thread(target=lambda:q.put(p.stdout.readline()),daemon=True).start()
 try:line=q.get(timeout=30)
 except queue.Empty:p.kill();p.wait();raise RuntimeError('launcher did not print URL')
 match=re.search(r'http://\S+',line)
 if not match:p.kill();p.wait();raise RuntimeError('launcher failed: '+line)
 return p,match.group(0)

with tempfile.TemporaryDirectory(prefix='smoke-',dir=os.environ.get('PORTFOLIO_SMOKE_TEMP')) as root:
 env=dict(os.environ,PORTFOLIO_DATA_DIR=root,PORTFOLIO_NO_BROWSER='1')
 p,url=launch(env)
 try:
  page=urllib.request.urlopen(url,timeout=10).read().decode()
  assert 'background:#0b1627' in page and 'ZgotmplZ' not in page
  if repo=='vanguard':
   fields={'policy':'apiVersion: vanguard/v1\nkind: SecurityPolicy\nthresholds: {critical: 0}\n','report':'{"findings":[{"id":"synthetic","severity":"critical"}]}'}
  else:fields={'action':'demo'}
  if repo=='creddrift':data=urllib.parse.urlencode(fields).encode();ct='application/x-www-form-urlencoded'
  else:data,ct=multipart(fields)
  req=urllib.request.Request(url,data=data,headers={'Content-Type':ct})
  page=urllib.request.urlopen(req,timeout=30).read().decode()
  assert 'ZgotmplZ' not in page
  if repo=='vanguard':assert '<h2>FAIL</h2>' in page
  elif repo=='logflow':assert '192.0.2.10' in page
  else:assert 'Potential secret' in page and 'synthetic-demo-only-abc123' not in page
  token=urllib.parse.parse_qs(urllib.parse.urlparse(url).query)['token'][0]
  export=urllib.parse.urljoin(url,'/export')+'?'+urllib.parse.urlencode({'token':token})
  payload=urllib.request.urlopen(export,timeout=10).read().decode();value=json.loads(payload)
  if repo=='vanguard':assert value['Passed'] is False and len(value['PolicyHash'])==64
  elif repo=='logflow':assert value['Report']['Complete'] and len(value['Report']['Alerts'])==1
  else:assert len(value)==1 and 'synthetic-demo-only-abc123' not in payload
  try:urllib.request.urlopen(url.split('?')[0],timeout=10);raise AssertionError('missing token allowed')
  except urllib.error.HTTPError as e:assert e.code==403
 finally:p.terminate();p.wait(timeout=15)
 # A force-killed process leaves its lock by design. Confirmed stopped above.
 for lock in Path(root).glob('*/application.lock'):lock.unlink()
 if repo!='vanguard':
  p,url=launch(env)
  try:
   page=urllib.request.urlopen(url,timeout=10).read().decode()
   assert ('192.0.2.10' if repo=='logflow' else 'Potential secret') in page
  finally:p.terminate();p.wait(timeout=15)
 print(repo+': packaged launch, rendered UI, demo/evaluation, export, token boundary, and persistence checks PASS')
