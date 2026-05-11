package service

import (
	"context"
	"fmt"
	"strings"
)

const (
	ExpireMonitorName = "Mikhmon-Expire-Monitor"

	ExpireMonitorComment = "Mikhmon Expire Monitor v2 [roskit]"
)

func (b *Bridge) DeployExpireMonitor(ctx context.Context, routerID, interval string) error {
	existing, err := b.CheckExpireMonitor(ctx, routerID)
	if err != nil {
		return err
	}

	script := GenerateExpireMonitorScript()

	if existing == nil {
		_, err = b.mutateAdd(ctx, routerID, "system/scheduler/add", map[string]string{
			"name":       ExpireMonitorName,
			"start-time": "00:00:00",
			"interval":   interval,
			"on-event":   script,
			"disabled":   "no",
			"comment":    ExpireMonitorComment,
		})
		return err
	}

	needsUpgrade := existing["comment"] != ExpireMonitorComment
	isDisabled := existing["disabled"] == "true" || existing["disabled"] == "yes"
	if needsUpgrade || isDisabled {
		_, err = b.Mutate(ctx, routerID, "system/scheduler/set",
			"=.id="+existing[".id"],
			"=interval="+interval,
			"=on-event="+script,
			"=comment="+ExpireMonitorComment,
			"=disabled=no",
		)
		return err
	}

	return nil
}

func (b *Bridge) RemoveExpireMonitor(ctx context.Context, routerID string) error {
	existing, err := b.CheckExpireMonitor(ctx, routerID)
	if err != nil || existing == nil {
		return err
	}
	_, err = b.Mutate(ctx, routerID, "system/scheduler/remove", "=.id="+existing[".id"])
	return err
}

func (b *Bridge) CheckExpireMonitor(ctx context.Context, routerID string) (map[string]string, error) {
	return b.FindSchedulerByName(ctx, routerID, ExpireMonitorName)
}

func (b *Bridge) SetSchedulerInterval(ctx context.Context, routerID, id, interval string) error {
	_, err := b.Mutate(ctx, routerID, "system/scheduler/set",
		"=.id="+id,
		"=interval="+interval,
	)
	return err
}

func GenerateExpireMonitorScript() string {
	return strings.ReplaceAll(expireMonitorScript, "\n", "; ")
}

const expireMonitorScript = `:local dateint do={
  :local montharray ("jan","feb","mar","apr","may","jun","jul","aug","sep","oct","nov","dec")
  :local days [:pick $d 4 6]
  :local month [:pick $d 0 3]
  :local year [:pick $d 7 11]
  :local monthint ([:find $montharray $month])
  :local month ($monthint + 1)
  :if ([len $month] = 1) do={
    :local zero ("0")
    :return [:tonum ("$year$zero$month$days")]
  } else={
    :return [:tonum ("$year$month$days")]
  }
}
:local timeint do={
  :local hours [:pick $t 0 2]
  :local minutes [:pick $t 3 5]
  :return ($hours * 60 + $minutes)
}
:local date [/system clock get date]
:local time [/system clock get time]
:local today [$dateint d=$date]
:local curtime [$timeint t=$time]
:local tyear [:pick $date 7 11]
:local lyear ($tyear - 1)
:foreach i in [/ip hotspot user find where comment~"/$tyear" || comment~"/$lyear"] do={
  :local comment [/ip hotspot user get $i comment]
  :local limit [/ip hotspot user get $i limit-uptime]
  :local name [/ip hotspot user get $i name]
  :local gettime [:pic $comment 12 20]
  :if ([:pic $comment 3] = "/" and [:pic $comment 6] = "/") do={
    :local expd [$dateint d=$comment]
    :local expt [$timeint t=$gettime]
    :if (
      ($expd < $today and $expt < $curtime) or
      ($expd < $today and $expt > $curtime) or
      ($expd = $today and $expt < $curtime)
      and $limit != "00:00:01"
    ) do={
      :if ([:pic $comment 21] = "N") do={
        [/ip hotspot user set limit-uptime=1s $i]
        [/ip hotspot active remove [find where user=$name]]
      } else={
        [/ip hotspot user remove $i]
        [/ip hotspot active remove [find where user=$name]]
      }
    }
  }
}`

var _ = fmt.Sprintf
