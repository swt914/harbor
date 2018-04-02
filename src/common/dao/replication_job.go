// Copyright (c) 2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dao

import (
	"time"

	"strings"

	"github.com/astaxie/beego/orm"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
)

// AddRepTarget ...
func AddRepTarget(target models.RepTarget) (int64, error) {
	o := GetOrmer()
	return o.Insert(&target)
}

// GetRepTarget ...
func GetRepTarget(id int64) (*models.RepTarget, error) {
	o := GetOrmer()
	t := models.RepTarget{ID: id}
	err := o.Read(&t)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// GetRepTargetByName ...
func GetRepTargetByName(name string) (*models.RepTarget, error) {
	o := GetOrmer()
	t := models.RepTarget{Name: name}
	err := o.Read(&t, "Name")
	if err == orm.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// GetRepTargetByEndpoint ...
func GetRepTargetByEndpoint(endpoint string) (*models.RepTarget, error) {
	o := GetOrmer()
	t := models.RepTarget{
		URL: endpoint,
	}
	err := o.Read(&t, "URL")
	if err == orm.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// DeleteRepTarget ...
func DeleteRepTarget(id int64) error {
	o := GetOrmer()
	_, err := o.Delete(&models.RepTarget{ID: id})
	return err
}

// UpdateRepTarget ...
func UpdateRepTarget(target models.RepTarget) error {
	o := GetOrmer()
	target.UpdateTime = time.Now()
	_, err := o.Update(&target, "URL", "Name", "Username", "Password", "Insecure", "UpdateTime")
	return err
}

// FilterRepTargets filters targets by name
func FilterRepTargets(name string) ([]*models.RepTarget, error) {
	o := GetOrmer()

	var args []interface{}

	sql := `select * from replication_target `
	if len(name) != 0 {
		sql += `where name like ? `
		args = append(args, `%`+Escape(name)+`%`)
	}
	sql += `order by creation_time`

	var targets []*models.RepTarget

	if _, err := o.Raw(sql, args).QueryRows(&targets); err != nil {
		return nil, err
	}

	return targets, nil
}

// AddRepPolicy ...
func AddRepPolicy(policy models.RepPolicy) (int64, error) {
	o := GetOrmer()
	sql := `insert into replication_policy (name, project_id, target_id, enabled, description, cron_str, creation_time, update_time, filters, replicate_deletion) 
				values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	params := []interface{}{}
	now := time.Now()
	params = append(params, policy.Name, policy.ProjectID, policy.TargetID, 1,
		policy.Description, policy.Trigger, now, now, policy.Filters,
		policy.ReplicateDeletion)

	result, err := o.Raw(sql, params...).Exec()
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetRepPolicy ...
func GetRepPolicy(id int64) (*models.RepPolicy, error) {
	o := GetOrmer()
	sql := `select * from replication_policy where id = ? and deleted = 0`

	var policy models.RepPolicy

	if err := o.Raw(sql, id).QueryRow(&policy); err != nil {
		if err == orm.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

// GetTotalOfRepPolicies returns the total count of replication policies
func GetTotalOfRepPolicies(name string, projectID int64) (int64, error) {
	qs := GetOrmer().QueryTable(&models.RepPolicy{}).Filter("deleted", 0)

	if len(name) != 0 {
		qs = qs.Filter("name__icontains", name)
	}

	if projectID != 0 {
		qs = qs.Filter("project_id", projectID)
	}

	return qs.Count()
}

// FilterRepPolicies filters policies by name and project ID
func FilterRepPolicies(name string, projectID, page, pageSize int64) ([]*models.RepPolicy, error) {
	o := GetOrmer()

	var args []interface{}

	sql := `select rp.id, rp.project_id, rp.target_id, 
				rt.name as target_name, rp.name, rp.description,
				rp.cron_str, rp.filters, rp.replicate_deletion, 
				rp.creation_time, rp.update_time, 
				count(rj.status) as error_job_count 
			from replication_policy rp 
			left join replication_target rt on rp.target_id=rt.id 
			left join replication_job rj on rp.id=rj.policy_id and (rj.status="error" 
				or rj.status="retrying") 
			where rp.deleted = 0 `

	if len(name) != 0 && projectID != 0 {
		sql += `and rp.name like ? and rp.project_id = ? `
		args = append(args, `%`+Escape(name)+`%`)
		args = append(args, projectID)
	} else if len(name) != 0 {
		sql += `and rp.name like ? `
		args = append(args, `%`+Escape(name)+`%`)
	} else if projectID != 0 {
		sql += `and rp.project_id = ? `
		args = append(args, projectID)
	}

	sql += `group by rp.id order by rp.creation_time`

	if page > 0 && pageSize > 0 {
		sql += ` limit ? offset ?`
		args = append(args, pageSize, (page-1)*pageSize)
	}

	var policies []*models.RepPolicy
	if _, err := o.Raw(sql, args).QueryRows(&policies); err != nil {
		return nil, err
	}

	return policies, nil
}

// GetRepPolicyByName ...
func GetRepPolicyByName(name string) (*models.RepPolicy, error) {
	o := GetOrmer()
	sql := `select * from replication_policy where deleted = 0 and name = ?`

	var policy models.RepPolicy

	if err := o.Raw(sql, name).QueryRow(&policy); err != nil {
		if err == orm.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

// GetRepPolicyByProject ...
func GetRepPolicyByProject(projectID int64) ([]*models.RepPolicy, error) {
	o := GetOrmer()
	sql := `select * from replication_policy where deleted = 0 and project_id = ?`

	var policies []*models.RepPolicy

	if _, err := o.Raw(sql, projectID).QueryRows(&policies); err != nil {
		return nil, err
	}

	return policies, nil
}

// GetRepPolicyByTarget ...
func GetRepPolicyByTarget(targetID int64) ([]*models.RepPolicy, error) {
	o := GetOrmer()
	sql := `select * from replication_policy where deleted = 0 and target_id = ?`

	var policies []*models.RepPolicy

	if _, err := o.Raw(sql, targetID).QueryRows(&policies); err != nil {
		return nil, err
	}

	return policies, nil
}

// GetRepPolicyByProjectAndTarget ...
func GetRepPolicyByProjectAndTarget(projectID, targetID int64) ([]*models.RepPolicy, error) {
	o := GetOrmer()
	sql := `select * from replication_policy where deleted = 0 and project_id = ? and target_id = ?`

	var policies []*models.RepPolicy

	if _, err := o.Raw(sql, projectID, targetID).QueryRows(&policies); err != nil {
		return nil, err
	}

	return policies, nil
}

// UpdateRepPolicy ...
func UpdateRepPolicy(policy *models.RepPolicy) error {
	o := GetOrmer()
	policy.UpdateTime = time.Now()
	_, err := o.Update(policy, "ProjectID", "TargetID", "Name", "Description",
		"Trigger", "Filters", "ReplicateDeletion", "UpdateTime")
	return err
}

// DeleteRepPolicy ...
func DeleteRepPolicy(id int64) error {
	o := GetOrmer()
	policy := &models.RepPolicy{
		ID:         id,
		Deleted:    1,
		UpdateTime: time.Now(),
	}
	_, err := o.Update(policy, "Deleted")
	return err
}

// AddRepJob ...
func AddRepJob(job models.RepJob) (int64, error) {
	o := GetOrmer()
	if len(job.Status) == 0 {
		job.Status = models.JobPending
	}
	if len(job.TagList) > 0 {
		job.Tags = strings.Join(job.TagList, ",")
	}
	return o.Insert(&job)
}

// GetRepJob ...
func GetRepJob(id int64) (*models.RepJob, error) {
	o := GetOrmer()
	j := models.RepJob{ID: id}
	err := o.Read(&j)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	genTagListForJob(&j)
	return &j, nil
}

// GetTotalCountOfRepJobs ...
func GetTotalCountOfRepJobs(query ...*models.RepJobQuery) (int64, error) {
	qs := repJobQueryConditions(query...)
	return qs.Count()
}

// GetRepJobs ...
func GetRepJobs(query ...*models.RepJobQuery) ([]*models.RepJob, error) {
	jobs := []*models.RepJob{}

	qs := repJobQueryConditions(query...)
	if len(query) > 0 && query[0] != nil {
		qs = paginateForQuerySetter(qs, query[0].Page, query[0].Size)
	}

	qs = qs.OrderBy("-UpdateTime")

	if _, err := qs.All(&jobs); err != nil {
		return jobs, err
	}

	genTagListForJob(jobs...)

	return jobs, nil
}

func repJobQueryConditions(query ...*models.RepJobQuery) orm.QuerySeter {
	qs := GetOrmer().QueryTable(new(models.RepJob))
	if len(query) == 0 || query[0] == nil {
		return qs
	}

	q := query[0]
	if q.PolicyID != 0 {
		qs = qs.Filter("PolicyID", q.PolicyID)
	}
	if len(q.Repository) > 0 {
		qs = qs.Filter("Repository__icontains", q.Repository)
	}
	if len(q.Statuses) > 0 {
		qs = qs.Filter("Status__in", q.Statuses)
	}
	if len(q.Operations) > 0 {
		qs = qs.Filter("Operation__in", q.Operations)
	}
	if q.StartTime != nil {
		qs = qs.Filter("CreationTime__gte", q.StartTime)
	}
	if q.EndTime != nil {
		qs = qs.Filter("CreationTime__lte", q.EndTime)
	}
	return qs
}

// DeleteRepJob ...
func DeleteRepJob(id int64) error {
	o := GetOrmer()
	_, err := o.Delete(&models.RepJob{ID: id})
	return err
}

// UpdateRepJobStatus ...
func UpdateRepJobStatus(id int64, status string) error {
	o := GetOrmer()
	j := models.RepJob{
		ID:         id,
		Status:     status,
		UpdateTime: time.Now(),
	}
	n, err := o.Update(&j, "Status", "UpdateTime")
	if n == 0 {
		log.Warningf("no records are updated when updating replication job %d", id)
	}
	return err
}

// SetRepJobUUID ...
func SetRepJobUUID(id int64, uuid string) error {
	o := GetOrmer()
	j := models.RepJob{
		ID:   id,
		UUID: uuid,
	}
	n, err := o.Update(&j, "UUID")
	if n == 0 {
		log.Warningf("no records are updated when updating replication job %d", id)
	}
	return err
}

func genTagListForJob(jobs ...*models.RepJob) {
	for _, j := range jobs {
		if len(j.Tags) > 0 {
			j.TagList = strings.Split(j.Tags, ",")
		}
	}
}
