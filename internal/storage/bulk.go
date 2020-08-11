package storage

import "github.com/jackc/pgx/v4"

// TODO benchmark

type chatRow struct {
	chatId, userId int64
}

type chatBulk struct {
	rows []chatRow
	idx  int
}

func (cb chatRow) toInterface() []interface{} {
	return []interface{}{cb.chatId, cb.userId}
}

func copyFromBulk(rows []chatRow) pgx.CopyFromSource {
	return &chatBulk{
		rows: rows,
		idx:  -1,
	}
}

func (cb *chatBulk) Next() bool {
	cb.idx++
	return cb.idx < len(cb.rows)
}

func (cb *chatBulk) Values() ([]interface{}, error) {
	return cb.rows[cb.idx].toInterface(), nil
}

func (cb *chatBulk) Err() error {
	return nil
}
