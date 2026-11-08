import html,json,os,re,subprocess,sys,tempfile,threading,queue,urllib.request,urllib.parse
from pathlib import Path
executable=sys.argv[1]
def multipart(fields):
 boundary='onboarding-smoke-boundary';data=b''
 for name,value in fields.items():data+=(f'--{boundary}\r\nContent-Disposition: form-data; name="{name}"\r\n\r\n{value}\r\n').encode()
 return data+f'--{boundary}--\r\n'.encode(),'multipart/form-data; boundary='+boundary
def launch(env):
 p=subprocess.Popen([str(Path(executable).resolve())],env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,encoding='utf-8');q=queue.Queue();threading.Thread(target=lambda:q.put(p.stdout.readline()),daemon=True).start()
 try:line=q.get(timeout=30)
 except queue.Empty:p.kill();p.wait();raise RuntimeError('no launcher URL')
 match=re.search(r'http://\S+',line)
 if not match:p.kill();p.wait();raise RuntimeError('launcher failed: '+line)
 return p,match.group(0)
def get(url):return urllib.request.urlopen(url,timeout=30).read().decode()
def post(url,fields,multi=False):
 if multi:data,ct=multipart(fields)
 else:data,ct=urllib.parse.urlencode(fields).encode(),'application/x-www-form-urlencoded'
 return urllib.request.urlopen(urllib.request.Request(url,data=data,headers={'Content-Type':ct}),timeout=30).read().decode()
with tempfile.TemporaryDirectory(prefix='logflow-',dir=os.environ.get('LOGFLOW_SMOKE_TEMP')) as root:
 env=dict(os.environ,LOGFLOW_DATA_DIR=root,LOGFLOW_NO_BROWSER='1')
 p,url=launch(env)
 try:
  page=get(url);assert 'Try the guided demo' in page and 'Use my data' in page and 'ZgotmplZ' not in page
  token=urllib.parse.parse_qs(urllib.parse.urlparse(url).query)['token'][0]
  def route(path):return urllib.parse.urljoin(url,path)+'?'+urllib.parse.urlencode({'token':token})
  setup=get(route('/setup'));assert 'Set up your first task' in setup
  assert not list(Path(root).glob('*/onboarding.json')), 'viewing setup marked guide complete'
  sample=get(route('/onboarding/sample')+'&name=example-logs.jsonl')
  records=[json.loads(line) for line in sample.splitlines()]
  assert len(records)==8 and all(record['message']=='Failed login attempt' for record in records)
  page=post(url+'&onboarding=demo',{'action':'demo'},True)
  assert 'id="results"' in page and 'href="#results"' in page
  assert 'Your first result is ready' in page and 'Guided demo - synthetic data' in page and 'ZgotmplZ' not in page
  assert '192.0.2.10' in page
  link=re.search(r'href="(/export[^\"]+)"',page);assert link,'missing export'
  payload=get(urllib.parse.urljoin(url,html.unescape(link.group(1))));value=json.loads(payload)
  assert value['Demo'] and value['Report']['Complete'] and len(value['Report']['Alerts'])==1
  assert len(value['Report']['Alerts'][0]['Evidence'])==6
  assert list(Path(root).glob('*/onboarding.json')),'completion not remembered'
  assert 'Try the guided demo' in get(route('/welcome'))
  assert json.loads(get(urllib.parse.urljoin(url,html.unescape(link.group(1)))))==value,'restart guide modified results'
  try:get(url.split('?')[0]);raise AssertionError('missing token allowed')
  except urllib.error.HTTPError as e:assert e.code==403
 finally:p.terminate();p.wait(timeout=15)
 for lock in Path(root).glob('*/application.lock'):lock.unlink()
 p,url=launch(env)
 try:
  page=get(url);assert 'Try the guided demo' not in page and 'Welcome back' in page
  assert '192.0.2.10' in page
 finally:p.terminate();p.wait(timeout=15)
 print('LogFlow: packaged welcome, setup, guided demo, next step, export, restart safety and remembered completion PASS')
