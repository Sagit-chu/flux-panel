package com.admin.service.impl;

import com.admin.common.dto.*;
import com.admin.common.lang.R;
import com.admin.entity.User;
import com.admin.entity.UserTunnel;
import com.admin.mapper.UserTunnelMapper;
import com.admin.service.UserService;
import com.admin.service.UserTunnelService;
import com.admin.service.ForwardService;
import com.admin.entity.Forward;
import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import org.springframework.beans.BeanUtils;
import org.springframework.context.annotation.Lazy;
import org.springframework.stereotype.Service;

import javax.annotation.Resource;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

@Service
public class UserTunnelServiceImpl extends ServiceImpl<UserTunnelMapper, UserTunnel> implements UserTunnelService {

    @Resource
    @Lazy
    private ForwardService forwardService;

    @Resource
    @Lazy
    private UserService userService;

    @Override
    public R assignUserTunnel(UserTunnelDto userTunnelDto) {
        int count = this.count(new QueryWrapper<UserTunnel>()
                .eq("user_id", userTunnelDto.getUserId())
                .eq("tunnel_id", userTunnelDto.getTunnelId()));
        if (count > 0) return R.err("该用户已拥有此隧道权限");

        User user = userService.getById(userTunnelDto.getUserId());
        if (user == null) return R.err("用户不存在");

        UserTunnel userTunnel = new UserTunnel();
        userTunnel.setUserId(userTunnelDto.getUserId());
        userTunnel.setTunnelId(userTunnelDto.getTunnelId());
        userTunnel.setSpeedId(userTunnelDto.getSpeedId());
        userTunnel.setStatus(1);
        userTunnel.setInFlow(0L);
        userTunnel.setOutFlow(0L);

        userTunnel.setFlow(userTunnelDto.getFlow() != null ? userTunnelDto.getFlow() : user.getFlow());
        userTunnel.setNum(userTunnelDto.getNum() != null ? userTunnelDto.getNum() : user.getNum());
        userTunnel.setFlowResetTime(userTunnelDto.getFlowResetTime() != null ? userTunnelDto.getFlowResetTime() : user.getFlowResetTime());
        userTunnel.setExpTime(userTunnelDto.getExpTime() != null ? userTunnelDto.getExpTime() : user.getExpTime());

        this.save(userTunnel);
        return R.ok();
    }

    @Override
    public R batchAssignUserTunnel(UserTunnelBatchAssignDto batchAssignDto) {
        User user = userService.getById(batchAssignDto.getUserId());
        if (user == null) return R.err("用户不存在");

        Map<Integer, UserTunnelBatchAssignDto.TunnelAssignItem> uniqueTunnels = new LinkedHashMap<>();
        for (UserTunnelBatchAssignDto.TunnelAssignItem item : batchAssignDto.getTunnels()) {
            uniqueTunnels.putIfAbsent(item.getTunnelId(), item);
        }

        Set<Integer> existingTunnelIds = this.list(
                new QueryWrapper<UserTunnel>()
                        .eq("user_id", batchAssignDto.getUserId())
                        .in("tunnel_id", uniqueTunnels.keySet())
        ).stream().map(UserTunnel::getTunnelId).collect(Collectors.toSet());

        List<UserTunnel> toSave = new ArrayList<>();
        List<Integer> skippedIds = new ArrayList<>();

        for (UserTunnelBatchAssignDto.TunnelAssignItem item : uniqueTunnels.values()) {
            if (existingTunnelIds.contains(item.getTunnelId())) {
                skippedIds.add(item.getTunnelId());
                continue;
            }

            UserTunnel ut = new UserTunnel();
            ut.setUserId(batchAssignDto.getUserId());
            ut.setTunnelId(item.getTunnelId());
            ut.setSpeedId(item.getSpeedId());
            ut.setStatus(1);
            ut.setInFlow(0L);
            ut.setOutFlow(0L);
            ut.setFlow(user.getFlow());
            ut.setNum(user.getNum());
            ut.setFlowResetTime(user.getFlowResetTime());
            ut.setExpTime(user.getExpTime());
            toSave.add(ut);
        }

        if (toSave.isEmpty()) {
            return R.err("所选隧道用户均已拥有权限");
        }

        this.saveBatch(toSave);

        if (!skippedIds.isEmpty()) {
            return R.ok("成功分配 " + toSave.size() + " 个隧道，跳过 " + skippedIds.size() + " 个已有权限的隧道");
        }
        return R.ok();
    }

    @Override
    public R getUserTunnelList(UserTunnelQueryDto queryDto) {
        List<UserTunnelWithDetailDto> userTunnelWithDetails = this.baseMapper.getUserTunnelWithDetails(queryDto.getUserId());
        return R.ok(userTunnelWithDetails);
    }

    @Override
    public R removeUserTunnel(Integer id) {
        UserTunnel userTunnel = this.getById(id);
        if (userTunnel == null) return R.err("未找到对应的用户隧道权限记录");

        List<Forward> forwardList = forwardService.list(new QueryWrapper<Forward>()
                .eq("user_id", userTunnel.getUserId())
                .eq("tunnel_id", userTunnel.getTunnelId()));
        for (Forward forward : forwardList) {
            forwardService.deleteForward(forward.getId());
        }
        this.removeById(id);
        return R.ok();
    }

    @Override
    public R updateUserTunnel(UserTunnelUpdateDto updateDto) {
        UserTunnel userTunnel = this.getById(updateDto.getId());
        if (userTunnel == null) return R.err("隧道不存在");
        boolean speedChanged = hasSpeedChanged(userTunnel.getSpeedId(), updateDto.getSpeedId());
        userTunnel.setFlow(updateDto.getFlow());
        userTunnel.setNum(updateDto.getNum());
        updateOptionalProperty(userTunnel::setFlowResetTime, updateDto.getFlowResetTime());
        updateOptionalProperty(userTunnel::setExpTime, updateDto.getExpTime());
        updateOptionalProperty(userTunnel::setStatus, updateDto.getStatus());
        userTunnel.setSpeedId(updateDto.getSpeedId());
        this.updateById(userTunnel);
        if (speedChanged) {
            List<Forward> forwardList = forwardService.list(new QueryWrapper<Forward>()
                    .eq("user_id", userTunnel.getUserId())
                    .eq("tunnel_id", userTunnel.getTunnelId()));
            for (Forward forward : forwardList) {
                ForwardUpdateDto forwardUpdateDto = new ForwardUpdateDto();
                forwardUpdateDto.setId(forward.getId());
                forwardUpdateDto.setUserId(forward.getUserId());
                forwardUpdateDto.setName(forward.getName());
                forwardUpdateDto.setRemoteAddr(forward.getRemoteAddr());
                forwardUpdateDto.setStrategy(forward.getStrategy());
                forwardService.updateForward(forwardUpdateDto);
            }
        }
        return R.ok();
    }

    private <T> void updateOptionalProperty(java.util.function.Consumer<T> setter, T value) {
        if (value != null) {
            setter.accept(value);
        }
    }

    private boolean hasSpeedChanged(Integer oldSpeedId, Integer newSpeedId) {
        if (oldSpeedId == null && newSpeedId == null) {
            return false;
        }
        if (oldSpeedId == null || newSpeedId == null) {
            return true;
        }
        return !oldSpeedId.equals(newSpeedId);
    }

}
