<?php
namespace App\Http\Controllers;
use App\Models\ProvisioningJob;
use App\Models\SshKey;
use App\Models\User;
use App\Services\ProvisioningService;
use Illuminate\Http\Request;
use Illuminate\Support\Str;
use RuntimeException;

class DashboardController extends Controller {
 private function user(): User { return User::findOrFail(session('people_user_id')); }
 public function index(){ $user=$this->user(); $storage=null; try{$storage=app(ProvisioningService::class)->storage($user);}catch(\Throwable $e){report($e);} return view('dashboard',['user'=>$user,'storage'=>$storage]); }
 public function keys(){return view('keys',['user'=>$this->user()]);}
 public function addKey(Request $request){
   $data=$request->validate(['name'=>['required','string','max:80'],'public_key'=>['required','string','max:8192']]);
   if(!preg_match('/^(ssh-(ed25519|rsa)|ecdsa-sha2-nistp256|sk-(ssh-ed25519|ecdsa-sha2-nistp256)@openssh\.com)\s+[^\s]+(?:\s+.*)?$/',$data['public_key'])) return back()->withErrors(['public_key'=>'Unsupported or malformed SSH public key.']);
   $fingerprint=hash('sha256',base64_decode(explode(' ',trim($data['public_key']))[1]??'',true) ?: $data['public_key']);
   $user=$this->user(); $key=$user->sshKeys()->create(['name'=>$data['name'],'public_key'=>trim($data['public_key']),'fingerprint'=>'SHA256:'.$fingerprint]);
   try{app(ProvisioningService::class)->addKey($user,$key->public_key);}catch(\Throwable $e){$key->delete(); return back()->withErrors(['public_key'=>'The provisioning agent rejected the key.']);}
   return back()->with('success','SSH key added.');
 }
 public function removeKey(SshKey $key){$user=$this->user(); abort_unless($key->user_id===$user->id,403); try{app(ProvisioningService::class)->removeKey($user,$key->public_key);$key->delete();return back()->with('success','SSH key removed.');}catch(\Throwable $e){report($e);return back()->withErrors(['key'=>'Could not remove the key.']);}}
 public function admin(){ $user=$this->user(); abort_unless($user->is_admin,403); return view('admin',['users'=>User::withCount('sshKeys')->latest()->get(),'jobs'=>ProvisioningJob::with('user')->latest()->limit(50)->get()]); }
 public function suspend(User $managed){$user=$this->user();abort_unless($user->is_admin,403);$managed->update(['status'=>'suspended']);return back()->with('success','User suspended.');}
 public function delete(User $managed){$user=$this->user();abort_unless($user->is_admin,403);app(ProvisioningService::class)->deleteUser($managed);$managed->update(['status'=>'deleted']);return back()->with('success','User deleted from the People host.');}
}
