package com.admin.common.dto;

import lombok.Data;
import javax.validation.Valid;
import javax.validation.constraints.NotNull;
import javax.validation.constraints.NotEmpty;
import java.util.List;

@Data
public class UserTunnelBatchAssignDto {
    
    @NotNull(message = "用户ID不能为空")
    private Integer userId;
    
    @Valid
    @NotEmpty(message = "隧道列表不能为空")
    private List<TunnelAssignItem> tunnels;
    
    @Data
    public static class TunnelAssignItem {
        @NotNull(message = "隧道ID不能为空")
        private Integer tunnelId;
        
        private Integer speedId;
    }
}
