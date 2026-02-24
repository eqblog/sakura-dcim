import React, { useEffect } from 'react';
import { Modal, Form, Input, Select, Switch } from 'antd';
import type { User, Role } from '../../types';

interface UserFormModalProps {
  open: boolean;
  user: User | null; // null = create mode
  roles: Role[];
  confirmLoading: boolean;
  onOk: (values: any) => void;
  onCancel: () => void;
}

const UserFormModal: React.FC<UserFormModalProps> = ({
  open,
  user,
  roles,
  confirmLoading,
  onOk,
  onCancel,
}) => {
  const [form] = Form.useForm();
  const isEdit = !!user;

  useEffect(() => {
    if (open) {
      form.resetFields();
      if (user) {
        form.setFieldsValue({
          email: user.email,
          name: user.name,
          role_id: user.role_id,
          is_active: user.is_active,
        });
      }
    }
  }, [open, user, form]);

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      onOk(values);
    } catch {
      // validation failed
    }
  };

  return (
    <Modal
      title={isEdit ? 'Edit User' : 'Create User'}
      open={open}
      onOk={handleOk}
      onCancel={onCancel}
      confirmLoading={confirmLoading}
      destroyOnClose
    >
      <Form form={form} layout="vertical" initialValues={{ is_active: true }}>
        <Form.Item
          name="email"
          label="Email"
          rules={[
            { required: true, message: 'Email is required' },
            { type: 'email', message: 'Invalid email' },
          ]}
        >
          <Input placeholder="user@example.com" />
        </Form.Item>
        <Form.Item
          name="name"
          label="Name"
          rules={[{ required: true, message: 'Name is required' }]}
        >
          <Input placeholder="Full name" />
        </Form.Item>
        <Form.Item
          name="password"
          label={isEdit ? 'Password (leave blank to keep current)' : 'Password'}
          rules={isEdit ? [] : [{ required: true, message: 'Password is required' }]}
        >
          <Input.Password placeholder={isEdit ? 'Unchanged' : 'Enter password'} />
        </Form.Item>
        <Form.Item name="role_id" label="Role">
          <Select
            placeholder="Select a role"
            allowClear
            options={roles.map((r) => ({ value: r.id, label: r.name }))}
          />
        </Form.Item>
        {isEdit && (
          <Form.Item name="is_active" label="Active" valuePropName="checked">
            <Switch />
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
};

export default UserFormModal;
