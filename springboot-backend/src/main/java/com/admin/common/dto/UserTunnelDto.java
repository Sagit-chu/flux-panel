package com.admin.common.dto;

import lombok.Data;
import javax.validation.constraints.NotNull;
import javax.validation.constraints.Min;

@Data
public class UserTunnelDto {
    
    @NotNull(message = "用户ID不能为空")
    private Integer userId;
    
    @NotNull(message = "隧道ID不能为空")
    private Integer tunnelId;
    
    @Min(value = 0, message = "流量限制不能小于0")
    private Long flow;
    
    @Min(value = 0, message = "转发数量不能小于0")
    private Integer num;
    
    private Long flowResetTime;
    
    private Long expTime;
    
    private Integer speedId;
}
