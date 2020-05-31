package system

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStore_GetUserStatus(t *testing.T) {
	type fields struct {
		path string
	}
	type args struct {
		username string
	}

	tempDir, err := ioutil.TempDir("", "open-bastion-testing")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	//Active
	err = os.MkdirAll(tempDir+"/alice", 0777)

	if err != nil {
		assert.Fail(t, err.Error())
	}

	f, err := os.Create(tempDir + "/alice/info.json")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	_, err = f.WriteString("{\"active\":true}")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	err = f.Close()

	if err != nil {
		assert.Fail(t, err.Error())
	}

	//Inactive
	err = os.MkdirAll(tempDir+"/bob", 0777)

	if err != nil {
		assert.Fail(t, err.Error())
	}

	f, err = os.Create(tempDir + "/bob/info.json")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	_, err = f.WriteString("{\"active\":false}")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	err = f.Close()

	if err != nil {
		assert.Fail(t, err.Error())
	}

	//Bad json
	err = os.MkdirAll(tempDir+"/diane", 0777)

	if err != nil {
		assert.Fail(t, err.Error())
	}

	f, err = os.Create(tempDir + "/diane/info.json")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	_, err = f.WriteString("{\"active\"false}")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	err = f.Close()

	if err != nil {
		assert.Fail(t, err.Error())
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "test active",
			fields: fields{
				path: tempDir,
			},
			args: args{
				username: "alice",
			},
			want:    Active,
			wantErr: false,
		},
		{
			name: "test inactive",
			fields: fields{
				path: tempDir,
			},
			args: args{
				username: "bob",
			},
			want:    Inactive,
			wantErr: false,
		},
		{
			name: "test not exist",
			fields: fields{
				path: tempDir,
			},
			args: args{
				username: "charlie",
			},
			want:    Error,
			wantErr: true,
		},
		{
			name: "bad json",
			fields: fields{
				path: tempDir,
			},
			args: args{
				username: "diane",
			},
			want:    Error,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Store{
				path: tt.fields.path,
			}
			got, err := s.GetUserStatus(tt.args.username)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}

	os.RemoveAll(tempDir)
}

func Test_isUsernameValid(t *testing.T) {
	type args struct {
		username string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test ok",
			args: args{
				username: "alice",
			},
			want: true,
		},
		{
			name: "test ok",
			args: args{
				username: "a$",
			},
			want: true,
		},
		{
			name: "test ok",
			args: args{
				username: "b-ob1",
			},
			want: true,
		},
		{
			name: "test fail",
			args: args{
				username: "$bob",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUsernameValid(tt.args.username)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStore_GetUserEgressPublicKey(t *testing.T) {
	type fields struct {
		path string
	}
	type args struct {
		username string
	}

	tempDir, err := ioutil.TempDir("", "open-bastion-testing")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	//Ok
	err = os.MkdirAll(tempDir+"/bob/egress-keys", 0777)

	if err != nil {
		assert.Fail(t, err.Error())
	}

	f, err := os.Create(tempDir + "/bob/egress-keys/bob.pub")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	_, err = f.WriteString("mysuperpublickey")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	err = f.Close()

	if err != nil {
		assert.Fail(t, err.Error())
	}

	//fail no file
	err = os.MkdirAll(tempDir+"/alice/egress-keys", 0777)

	if err != nil {
		assert.Fail(t, err.Error())
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "test ok",
			fields: fields{
				path: tempDir,
			},
			args: args{
				username: "bob",
			},
			want:    []byte("mysuperpublickey"),
			wantErr: false,
		},
		{
			name: "test fail no file",
			fields: fields{
				path: tempDir,
			},
			args: args{
				username: "alice",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Store{
				path: tt.fields.path,
			}
			got, err := s.GetUserEgressPublicKey(tt.args.username)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}

	os.RemoveAll(tempDir)
}

func TestStore_GetUserEgressPrivateKey(t *testing.T) {
	type fields struct {
		path string
	}
	type args struct {
		username string
	}

	tempDir, err := ioutil.TempDir("", "open-bastion-testing")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	//Ok
	err = os.MkdirAll(tempDir+"/bob/egress-keys", 0777)

	if err != nil {
		assert.Fail(t, err.Error())
	}

	f, err := os.Create(tempDir + "/bob/egress-keys/bob")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	_, err = f.WriteString("mysuperprivatekey")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	err = f.Close()

	if err != nil {
		assert.Fail(t, err.Error())
	}

	//fail no file
	err = os.MkdirAll(tempDir+"/alice/egress-keys", 0777)

	if err != nil {
		assert.Fail(t, err.Error())
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "test ok",
			fields: fields{
				path: tempDir,
			},
			args: args{
				username: "bob",
			},
			want:    []byte("mysuperprivatekey"),
			wantErr: false,
		},
		{
			name: "test fail no file",
			fields: fields{
				path: tempDir,
			},
			args: args{
				username: "alice",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Store{
				path: tt.fields.path,
			}
			got, err := s.GetUserEgressPrivateKey(tt.args.username)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}

	os.RemoveAll(tempDir)
}

func TestStore_DeleteUser(t *testing.T) {
	type fields struct {
		path string
	}
	type args struct {
		username string
	}

	tempDir, err := ioutil.TempDir("", "open-bastion-testing")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	//Ok
	err = os.MkdirAll(tempDir+"/bob/egress-keys", 0777)

	if err != nil {
		assert.Fail(t, err.Error())
	}

	f, err := os.Create(tempDir + "/bob/egress-keys/bob")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	_, err = f.WriteString("mysuperprivatekey")

	if err != nil {
		assert.Fail(t, err.Error())
	}

	err = f.Close()

	if err != nil {
		assert.Fail(t, err.Error())
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test ok",
			fields: fields{
				path: tempDir,
			},
			args: args{
				username: "bob",
			},
			wantErr: false,
		},
		{
			name: "ok no file",
			fields: fields{
				path: tempDir,
			},
			args: args{
				username: "alice",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Store{
				path: tt.fields.path,
			}

			err := s.DeleteUser(tt.args.username)

			assert.Equal(t, tt.wantErr, err != nil)

			if _, err := os.Stat(tempDir + "/" + tt.args.username); os.IsExist(err) {
				assert.Fail(t, "error, file not deleted")
			}
		})
	}

	os.RemoveAll(tempDir)
}
