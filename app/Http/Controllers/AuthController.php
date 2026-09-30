<?php
namespace App\Http\Controllers;
use App\Models\User;
use App\Services\OidcService;
use App\Services\ProvisioningService;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Str;
use Throwable;

class AuthController extends Controller {
 public function login(){ return app(OidcService::class)->authenticate() ? redirect()->route('dashboard') : redirect()->route('auth.login'); }
 public function logout(Request $request): RedirectResponse { $request->session()->invalidate(); $request->session()->regenerateToken(); return redirect()->route('home'); }
 public function callback(){ return redirect()->route('dashboard'); }
 public function loginFromOidc(): RedirectResponse {
   try {
    $claims=app(OidcService::class)->authenticate();
    $username=Str::lower($claims['preferred_username'] ?? $claims['nickname'] ?? '');
    if(!preg_match('/^[a-z_][a-z0-9_-]{0,31}$/',$username)) abort(403,'Your Authentik username is not valid for a Linux account.');
    $groups=$claims['groups'] ?? [];
    if(is_string($groups)) $groups=[$groups];
    if(!in_array(config('people.authentik_group'),$groups,true)) abort(403,'You are not a member of the People group.');
    $user=User::firstOrCreate(['authentik_subject'=>$claims['sub']],['username'=>$username,'email'=>$claims['email'] ?? null,'display_name'=>$claims['name'] ?? $username,'status'=>'provisioning','is_admin'=>in_array(config('people.authentik_admin_group'),$groups,true)]);
    $user->update(['email'=>$claims['email'] ?? $user->email,'display_name'=>$claims['name'] ?? $user->display_name,'is_admin'=>in_array(config('people.authentik_admin_group'),$groups,true)]);
    if($user->status==='provisioning') { app(ProvisioningService::class)->createUser($user); $user->update(['status'=>'active']); }
    session(['people_user_id'=>$user->id]);
    session()->regenerate();
    return redirect()->route('dashboard');
   } catch(Throwable $e) { report($e); return redirect()->route('home')->with('error','Login failed. Check the dashboard logs.'); }
 }
}
