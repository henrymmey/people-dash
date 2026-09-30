<?php
namespace App\Services;
use App\Models\ProvisioningJob;
use App\Models\User;
use Illuminate\Support\Facades\Http;
use RuntimeException;
class ProvisioningService {
 private function client(){return Http::baseUrl(rtrim(config('people.agent_url'),'/'))->withToken(config('people.agent_token'))->acceptJson()->asJson()->timeout(15);}
 public function createUser(User $user): ProvisioningJob{return $this->run($user,'CREATE_USER',['username'=>$user->username,'email'=>$user->email]);}
 public function deleteUser(User $user): ProvisioningJob{return $this->run($user,'DELETE_USER',['username'=>$user->username]);}
 public function addKey(User $user,string $publicKey): ProvisioningJob{return $this->run($user,'ADD_SSH_KEY',['username'=>$user->username,'public_key'=>$publicKey]);}
 public function removeKey(User $user,string $publicKey): ProvisioningJob{return $this->run($user,'REMOVE_SSH_KEY',['username'=>$user->username,'public_key'=>$publicKey]);}
 public function storage(User $user): array{$r=$this->client()->get('/v1/users/'.rawurlencode($user->username).'/storage');if(!$r->successful())throw new RuntimeException('Provisioning agent storage request failed.');return $r->json();}
 private function run(User $user,string $type,array $payload): ProvisioningJob{
  $job=ProvisioningJob::create(['user_id'=>$user->id,'type'=>$type,'status'=>'running','payload'=>$payload]);
  try{
   $path=match($type){'CREATE_USER'=>'/v1/users','DELETE_USER'=>'/v1/users/'.rawurlencode($user->username),'ADD_SSH_KEY'=>'/v1/users/'.rawurlencode($user->username).'/ssh-keys','REMOVE_SSH_KEY'=>'/v1/users/'.rawurlencode($user->username).'/ssh-keys'};
   $response=match($type){'CREATE_USER'=>$this->client()->post($path,$payload),'DELETE_USER'=>$this->client()->delete($path),'ADD_SSH_KEY'=>$this->client()->post($path,['public_key'=>$payload['public_key']]),'REMOVE_SSH_KEY'=>$this->client()->withBody(json_encode(['public_key'=>$payload['public_key']],JSON_THROW_ON_ERROR),'application/json')->delete($path)};
   if(!$response->successful())throw new RuntimeException('Agent returned HTTP '.$response->status().': '.$response->body());
   $job->update(['status'=>'completed','completed_at'=>now()]);return $job;
  }catch(\Throwable $e){$job->update(['status'=>'failed','error'=>mb_substr($e->getMessage(),0,2000),'completed_at'=>now()]);throw $e;}
 }
}
