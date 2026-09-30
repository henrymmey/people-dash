<?php

namespace App\Http\Controllers;

use App\Models\ProvisioningJob;
use App\Models\SshKey;
use App\Models\User;
use App\Services\ProvisioningService;
use Illuminate\Database\QueryException;
use Illuminate\Http\Request;

class DashboardController extends Controller
{
    private function user(): User { return User::findOrFail(session('people_user_id')); }
    public function index() { $user=$this->user(); $storage=null; try{$storage=app(ProvisioningService::class)->storage($user);}catch(\Throwable $e){report($e);} return view('dashboard',['user'=>$user,'storage'=>$storage]); }
    public function keys() { return view('keys',['user'=>$this->user()]); }
    public function addKey(Request $request) {
        $data=$request->validate(['name'=>['required','string','max:80'],'public_key'=>['required','string','max:8192']]); $publicKey=trim($data['public_key']);
        if(!preg_match('/^(ssh-(ed25519|rsa)|ecdsa-sha2-nistp256|sk-(ssh-ed25519|ecdsa-sha2-nistp256)@openssh\.com)\s+[^\s]+(?:\s+.*)?$/',$publicKey)) return back()->withErrors(['public_key'=>'Unsupported or malformed SSH public key.']);
        $parts=preg_split('/\s+/',$publicKey,3); $decoded=isset($parts[1])?base64_decode($parts[1],true):false; if($decoded===false) return back()->withErrors(['public_key'=>'The public key encoding is invalid.']);
        $fingerprint='SHA256:'.rtrim(strtr(base64_encode(hash('sha256',$decoded,true)), '+/','-_'),'='); $user=$this->user();
        if($user->sshKeys()->where('fingerprint',$fingerprint)->exists()) return back()->withErrors(['public_key'=>'This SSH key is already registered.']);
        try{$key=$user->sshKeys()->create(['name'=>$data['name'],'public_key'=>$publicKey,'fingerprint'=>$fingerprint]);}catch(QueryException $e){report($e);return back()->withErrors(['public_key'=>'This SSH key is already registered.']);}
        try{app(ProvisioningService::class)->addKey($user,$key->public_key);}catch(\Throwable $e){report($e);$key->delete();return back()->withErrors(['public_key'=>'The provisioning agent rejected the key.']);}
        return back()->with('success','SSH key added.');
    }
    public function removeKey(SshKey $key) { $user=$this->user(); abort_unless($key->user_id===$user->id,403); try{app(ProvisioningService::class)->removeKey($user,$key->public_key);$key->delete();return back()->with('success','SSH key removed.');}catch(\Throwable $e){report($e);return back()->withErrors(['key'=>'Could not remove the key.']);} }
    public function admin() { $user=$this->user(); abort_unless($user->is_admin,403); return view('admin',['users'=>User::withCount('sshKeys')->latest()->get(),'jobs'=>ProvisioningJob::with('user')->latest()->limit(50)->get()]); }
    public function suspend(User $managed) { $user=$this->user(); abort_unless($user->is_admin,403); abort_if($managed->id===$user->id,422,'You cannot suspend your own account.'); if($managed->status==='deleted')return back()->withErrors(['user'=>'Deleted accounts cannot be suspended.']); try{app(ProvisioningService::class)->suspendUser($managed);$managed->update(['status'=>'suspended']);return back()->with('success',"User {$managed->username} suspended.");}catch(\Throwable $e){report($e);return back()->withErrors(['user'=>'The People host could not suspend this user.']);} }
    public function resume(User $managed) { $user=$this->user(); abort_unless($user->is_admin,403); abort_if($managed->status==='deleted',422,'Deleted accounts cannot be resumed.'); try{app(ProvisioningService::class)->resumeUser($managed);$managed->update(['status'=>'active']);return back()->with('success',"User {$managed->username} resumed.");}catch(\Throwable $e){report($e);return back()->withErrors(['user'=>'The People host could not resume this user.']);} }
    public function delete(User $managed) { $user=$this->user(); abort_unless($user->is_admin,403); abort_if($managed->id===$user->id,422,'You cannot delete your own account.'); if($managed->status==='deleted')return back()->withErrors(['user'=>'This account is already deleted.']); try{app(ProvisioningService::class)->deleteUser($managed);$managed->update(['status'=>'deleted']);return back()->with('success',"User {$managed->username} deleted from the People host.");}catch(\Throwable $e){report($e);return back()->withErrors(['user'=>'The People host could not delete this user.']);} }
}
