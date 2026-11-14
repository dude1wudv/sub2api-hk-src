//go:build integration

package repository

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChatMigrationConstraints(t *testing.T) {
	tx := testTx(t)
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := tx.Exec(query, args...)
		require.NoError(t, err)
	}
	reject := func(query string, args ...any) {
		t.Helper()
		exec("SAVEPOINT chat_constraint")
		_, err := tx.Exec(query, args...)
		require.Error(t, err)
		exec("ROLLBACK TO SAVEPOINT chat_constraint")
		exec("RELEASE SAVEPOINT chat_constraint")
	}
	newUser := func() int64 {
		var id int64
		require.NoError(t, tx.QueryRow("INSERT INTO users(email,password_hash) VALUES($1,'chat-test-only') RETURNING id", uuid.NewString()+"@chat.invalid").Scan(&id))
		return id
	}
	owner, other := newUser(), newUser()
	var groupID int64
	require.NoError(t, tx.QueryRow("INSERT INTO groups(name,platform) VALUES($1,'openai') RETURNING id", uuid.NewString()).Scan(&groupID))
	var policy sql.NullString
	require.NoError(t, tx.QueryRow("SELECT chat_model_policies::text FROM groups WHERE id=$1", groupID).Scan(&policy))
	require.False(t, policy.Valid)
	preset := uuid.New()
	exec("INSERT INTO chat_model_presets(id,user_id,group_id,model,is_default) VALUES($1,$2,$3,'same-model',true)", preset, owner, groupID)
	var normalEffort, thinkingEffort string
	require.NoError(t, tx.QueryRow("SELECT normal_reasoning_effort,thinking_reasoning_effort FROM chat_model_presets WHERE id=$1", preset).Scan(&normalEffort, &thinkingEffort))
	require.Equal(t, "low", normalEffort)
	require.Equal(t, "high", thinkingEffort)
	reject("UPDATE chat_model_presets SET normal_reasoning_effort='none' WHERE id=$1", preset)
	reject("UPDATE chat_model_presets SET thinking_reasoning_effort='max' WHERE id=$1", preset)
	reject("INSERT INTO chat_model_presets(id,user_id,group_id,model) VALUES($1,$2,$3,'same-model')", uuid.New(), owner, groupID)
	reject("INSERT INTO chat_model_presets(id,user_id,group_id,model,is_default) VALUES($1,$2,$3,'different',true)", uuid.New(), owner, groupID)
	exec("INSERT INTO chat_model_presets(id,user_id,group_id,model,is_default) VALUES($1,$2,$3,'same-model',true)", uuid.New(), other, groupID)
	var keyID int64
	require.NoError(t, tx.QueryRow("INSERT INTO api_keys(user_id,key,name,group_id) VALUES($1,$2,'chat-test-only',$3) RETURNING id", owner, uuid.NewString(), groupID).Scan(&keyID))
	exec("INSERT INTO chat_credentials(user_id,group_id,api_key_id) VALUES($1,$2,$3)", owner, groupID, keyID)
	reject("INSERT INTO chat_credentials(user_id,group_id,api_key_id) VALUES($1,$2,$3)", owner, groupID, keyID)
	reject("INSERT INTO chat_credentials(user_id,group_id,api_key_id) VALUES($1,$2,$3)", other, groupID, keyID)

	session, otherSession, attachment := uuid.New(), uuid.New(), uuid.New()
	snapshot, err := json.Marshal(map[string]any{
		"messages": []any{
			map[string]any{"id": "u", "role": "user", "content": "原始内容", "selectedChildId": "a"},
			map[string]any{"id": "a", "parentId": "u", "blocks": []any{map[string]string{"type": "thinking", "thinking": "思考"}, map[string]string{"type": "text", "text": "回答"}}},
			map[string]any{"id": "b", "parentId": "u", "blocks": []any{}},
		},
	})
	require.NoError(t, err)
	exec("INSERT INTO chat_sessions(id,user_id,snapshot) VALUES($1,$2,$3::jsonb)", session, owner, string(snapshot))
	exec("INSERT INTO chat_sessions(id,user_id,snapshot) VALUES($1,$2,'{}')", otherSession, other)
	var stored string
	require.NoError(t, tx.QueryRow("SELECT snapshot::text FROM chat_sessions WHERE user_id=$1 AND id=$2", owner, session).Scan(&stored))
	require.JSONEq(t, string(snapshot), stored)
	exec("INSERT INTO chat_attachments(id,user_id,storage_key,original_name,byte_size,purpose,status) VALUES($1,$2,$3,'test.png',4,'image','ready')", attachment, owner, uuid.NewString())
	exec("INSERT INTO chat_session_attachments(user_id,session_id,attachment_id) VALUES($1,$2,$3)", owner, session, attachment)
	reject("INSERT INTO chat_session_attachments(user_id,session_id,attachment_id) VALUES($1,$2,$3)", other, otherSession, attachment)
	reject("INSERT INTO chat_session_attachments(user_id,session_id,attachment_id) VALUES($1,$2,$3)", owner, otherSession, attachment)
	reject("DELETE FROM chat_attachments WHERE id=$1", attachment)
	reject("INSERT INTO chat_attachments(id,user_id,storage_key,original_name,byte_size,purpose) VALUES($1,$2,$3,'too-large.txt',33554433,'user_data')", uuid.New(), owner, uuid.NewString())
	run, request, instance := uuid.New(), uuid.New(), uuid.New()
	insertRun := "INSERT INTO chat_generation_runs(id,user_id,session_id,assistant_message_id,preset_id,group_id,model,request_id,operation,owner_instance_id,lease_expires_at,started_at,heartbeat_at) VALUES($1,$2,$3,'a',$4,$5,'same-model',$6,'send',$7,NOW()+INTERVAL '60 seconds',NOW(),NOW())"
	exec(insertRun, run, owner, session, preset, groupID, request, instance)
	reject(insertRun, uuid.New(), owner, session, preset, groupID, uuid.New(), instance)
	reject(insertRun, uuid.New(), other, session, preset, groupID, uuid.New(), instance)
	exec("UPDATE chat_generation_runs SET status='stopped',ended_at=NOW() WHERE id=$1", run)
	reject(insertRun, uuid.New(), owner, session, preset, groupID, request, instance)
	exec(insertRun, uuid.New(), owner, session, preset, groupID, uuid.New(), instance)
	exec("DELETE FROM chat_model_presets WHERE id=$1", preset)
	var runs int
	require.NoError(t, tx.QueryRow("SELECT count(*) FROM chat_generation_runs WHERE user_id=$1", owner).Scan(&runs))
	require.Equal(t, 2, runs)
	for _, want := range []int64{1, 0} {
		result, err := tx.Exec("UPDATE chat_sessions SET revision=revision+1 WHERE user_id=$1 AND id=$2 AND revision=0", owner, session)
		require.NoError(t, err)
		changed, err := result.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, want, changed)
	}
}
