<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

class User extends Model
{
    protected $fillable = ['authentik_subject','username','email','display_name','status','is_admin'];
    protected $casts = ['is_admin'=>'boolean'];
    public function sshKeys(): HasMany { return $this->hasMany(SshKey::class); }
    public function jobs(): HasMany { return $this->hasMany(ProvisioningJob::class); }
}
