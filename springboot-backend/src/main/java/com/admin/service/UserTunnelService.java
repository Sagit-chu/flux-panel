package com.admin.service;

import com.admin.common.dto.UserTunnelBatchAssignDto;
import com.admin.common.dto.UserTunnelDto;
import com.admin.common.dto.UserTunnelQueryDto;
import com.admin.common.dto.UserTunnelUpdateDto;
import com.admin.common.lang.R;
import com.admin.entity.UserTunnel;
import com.baomidou.mybatisplus.extension.service.IService;

public interface UserTunnelService extends IService<UserTunnel> {

    R assignUserTunnel(UserTunnelDto userTunnelDto);
    
    R batchAssignUserTunnel(UserTunnelBatchAssignDto batchAssignDto);
    
    R getUserTunnelList(UserTunnelQueryDto queryDto);
    
    R removeUserTunnel(Integer id);
    
    R updateUserTunnel(UserTunnelUpdateDto updateDto);

}
