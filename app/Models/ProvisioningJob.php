<?php
namespace App\Models;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
class ProvisioningJob extends Model {
    protected $fillable=['user_id','type','status','payload','error','completed_at'];
    protected $casts=['payload'=>'array','completed_at'=>'datetime'];
    public function user(): BelongsTo { return $this->belongsTo(User::class); }
}
